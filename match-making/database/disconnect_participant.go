package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making_constants "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications_constants "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/constants"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// DisconnectParticipant disconnects a participant from all non-persistent gatherings
func DisconnectParticipant(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection) {
	var nexError *nex.Error

	rows, err := manager.Database.Query(`SELECT id, owner_pid, host_pid, min_participants, max_participants, participation_policy, policy_argument, flags, state, description, type, participants FROM matchmaking.gatherings WHERE $1 = ANY(participants) AND registered=true`, connection.PID())
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return
	}

	for rows.Next() {
		gathering := match_making_types.NewGathering()
		var gatheringType string
		var participants []uint64

		err = rows.Scan(
			&gathering.ID,
			&gathering.OwnerPID,
			&gathering.HostPID,
			&gathering.MinimumParticipants,
			&gathering.MaximumParticipants,
			&gathering.ParticipationPolicy,
			&gathering.PolicyArgument,
			&gathering.Flags,
			&gathering.State,
			&gathering.Description,
			&gatheringType,
			pqextended.Array(&participants),
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		// * If the gathering is a PersistentGathering and the gathering isn't set to leave when disconnecting, ignore and continue
		//
		// TODO - Is the match_making.GatheringFlags.PersistentGathering check correct here?
		if gathering.Flags.HasFlag(match_making_constants.GatheringFlagPersistentGathering) || (gatheringType == "PersistentGathering" && !gathering.Flags.HasFlag(match_making_constants.GatheringFlagNoPersistentParticipation)) {
			continue
		}

		// * Since the participant is leaving, override the participant list to avoid sending notifications to them
		participants, nexError = RemoveParticipantFromGathering(manager, uint32(gathering.ID), uint64(connection.PID()))
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			continue
		}

		nexError = tracking.LogDisconnectGathering(manager.Database, connection.PID(), uint32(gathering.ID), participants)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			continue
		}

		// * If the gathering is a persistent gathering and allows zero users, only remove the participant from the gathering
		if gathering.Flags.HasFlag(match_making_constants.GatheringFlagPersistentGathering) || gathering.Flags.HasFlag(match_making_constants.GatheringFlagAllowNoParticipant) {
			continue
		}

		if len(participants) == 0 {
			// * There are no more participants, so we only have to unregister the gathering
			// * Since the participant is disconnecting, we don't send notification events
			nexError = UnregisterGathering(manager, connection.PID(), uint32(gathering.ID))
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
			}

			continue
		}

		ownerPID := uint64(gathering.OwnerPID)
		if connection.PID().Equals(gathering.OwnerPID) {
			// * This flag tells the server to change the matchmake session owner if they disconnect
			// * If the flag is not set, delete the session
			// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
			if !gathering.Flags.HasFlag(match_making_constants.GatheringFlagMigrateOwner) {
				nexError = UnregisterGathering(manager, connection.PID(), uint32(gathering.ID))
				if nexError != nil {
					common_globals.Logger.Error(nexError.Error())
					continue
				}

				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID().Copy().(types.PID)
				oEvent.Type = notifications_constants.NotificationCategoryGatheringUnregistered.Build()
				oEvent.Param1 = types.UInt64(gathering.ID)

				common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, common_globals.RemoveDuplicates(participants))

				continue
			}

			ownerPID, nexError = MigrateGatheringOwnership(manager, connection, gathering, participants)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
			}
		}

		// * If the host has disconnected, set the owner as the new host. We can guarantee that the ownerPID is not zero,
		// * since otherwise the gathering would have been unregistered by MigrateGatheringOwnership
		if connection.PID().Equals(gathering.HostPID) && ownerPID != 0 {
			nexError = UpdateSessionHost(manager, uint32(gathering.ID), types.NewPID(ownerPID), types.NewPID(ownerPID))
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
			} else {
				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID().Copy().(types.PID)
				oEvent.Type = notifications_constants.NotificationCategoryHostChangeEvent.Build()
				oEvent.Param1 = types.UInt64(gathering.ID)

				// TODO - Should the notification actually be sent to all participants?
				common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, common_globals.RemoveDuplicates(participants))

				nexError = tracking.LogChangeHost(manager.Database, connection.PID(), uint32(gathering.ID), gathering.HostPID, types.NewPID(ownerPID))
				if nexError != nil {
					common_globals.Logger.Error(nexError.Error())
					continue
				}
			}
		}

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID().Copy().(types.PID)
		oEvent.Type = notifications_constants.NotificationCategoryParticipationEvent.Build(notifications_constants.ParticipationEventsDisconnect)
		oEvent.Param1 = types.UInt64(gathering.ID)
		oEvent.Param2 = types.UInt64(connection.PID())

		var participantEndedTargets []uint64

		// * When the VerboseParticipants or VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
		if gathering.Flags.HasFlag(match_making_constants.GatheringFlagNotifyParticipationEventsToAllParticipants) || gathering.Flags.HasFlag(match_making_constants.GatheringFlagNotifyParticipationEventsToAllParticipantsReproducibly) {
			participantEndedTargets = common_globals.RemoveDuplicates(participants)
		} else {
			participantEndedTargets = []uint64{uint64(gathering.OwnerPID)}
		}

		// * Only send the notification event to the owner
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participantEndedTargets)
	}

	rows.Close()
}
