package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// DisconnectParticipant disconnects a participant from all non-persistent gatherings
func DisconnectParticipant(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection) {
	var nexError *nex.Error

	rows, err := manager.Database.Query(`SELECT id, owner_pid, host_pid, min_participants, max_participants, participation_policy, policy_argument, flags, state, description, type, participants FROM matchmaking.gatherings WHERE $1 = ANY(participants) AND registered=true`, connection.PID().Value())
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return
	}

	for rows.Next() {
		var gatheringID uint32
		var ownerPID uint64
		var hostPID uint64
		var minParticipants uint16
		var maxParticipants uint16
		var participationPolicy uint32
		var policyArgument uint32
		var flags uint32
		var state uint32
		var description string
		var gatheringType string
		var participants []uint64

		err = rows.Scan(
			&gatheringID,
			&ownerPID,
			&hostPID,
			&minParticipants,
			&maxParticipants,
			&participationPolicy,
			&policyArgument,
			&flags,
			&state,
			&description,
			&gatheringType,
			pqextended.Array(&participants),
		)
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			continue
		}

		// * If the gathering is a PersistentGathering and the gathering isn't set to leave when disconnecting, ignore and continue
		if gatheringType == "PersistentGathering" && flags & match_making.GatheringFlags.PersistentGatheringLeaveParticipation == 0 {
			continue
		}

		gathering := match_making_types.NewGathering()
		gathering.ID.Value = gatheringID
		gathering.OwnerPID = types.NewPID(ownerPID)
		gathering.HostPID = types.NewPID(hostPID)
		gathering.MinimumParticipants.Value = minParticipants
		gathering.MaximumParticipants.Value = maxParticipants
		gathering.ParticipationPolicy.Value = participationPolicy
		gathering.PolicyArgument.Value = policyArgument
		gathering.Flags.Value = flags
		gathering.State.Value = state
		gathering.Description.Value = description

		// * Since the participant is leaving, override the participant list to avoid sending notifications to them
		participants, nexError = RemoveParticipantFromGathering(manager, gatheringID, connection.PID().Value())
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			continue
		}

		nexError = tracking.LogDisconnectGathering(manager.Database, connection.PID(), gatheringID, participants)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			continue
		}

		if len(participants) == 0 {
			// * There are no more participants, so we only have to unregister the gathering
			// * Since the participant is disconnecting, we don't send notification events
			nexError = UnregisterGathering(manager, connection.PID(), gatheringID)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
			}

			continue
		}

		if connection.PID().Equals(gathering.OwnerPID) {
			// * This flag tells the server to change the matchmake session owner if they disconnect
			// * If the flag is not set, delete the session
			// * More info: https://nintendo-wiki.pretendo.network/docs/nex/protocols/match-making/types#flags
			if gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) == 0 {
				nexError = UnregisterGathering(manager, connection.PID(), gatheringID)
				if nexError != nil {
					common_globals.Logger.Error(nexError.Error())
					continue
				}

				category := notifications.NotificationCategories.GatheringUnregistered
				subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID().Copy().(*types.PID)
				oEvent.Type.Value = notifications.BuildNotificationType(category, subtype)
				oEvent.Param1.Value = gatheringID

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
			nexError = UpdateSessionHost(manager, gatheringID, types.NewPID(ownerPID), types.NewPID(ownerPID))
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
			} else {
				category := notifications.NotificationCategories.HostChanged
				subtype := notifications.NotificationSubTypes.HostChanged.None

				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID().Copy().(*types.PID)
				oEvent.Type.Value = notifications.BuildNotificationType(category, subtype)
				oEvent.Param1.Value = gatheringID

				// TODO - Should the notification actually be sent to all participants?
				common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, common_globals.RemoveDuplicates(participants))

				nexError = tracking.LogChangeHost(manager.Database, connection.PID(), gatheringID, gathering.HostPID, types.NewPID(ownerPID))
				if nexError != nil {
					common_globals.Logger.Error(nexError.Error())
					continue
				}
			}
		}

		category := notifications.NotificationCategories.Participation
		subtype := notifications.NotificationSubTypes.Participation.Disconnected

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID().Copy().(*types.PID)
		oEvent.Type = types.NewPrimitiveU32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1.Value = gatheringID
		oEvent.Param2.Value = connection.PID().LegacyValue() // TODO - This assumes a legacy client. Will not work on the Switch

		var participantEndedTargets []uint64

		// * When the VerboseParticipants or VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
		if gathering.Flags.PAND(match_making.GatheringFlags.VerboseParticipants | match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
			participantEndedTargets = common_globals.RemoveDuplicates(participants)
		} else {
			participantEndedTargets = []uint64{gathering.OwnerPID.Value()}
		}

		// * Only send the notification event to the owner
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participantEndedTargets)
	}

	rows.Close()
}
