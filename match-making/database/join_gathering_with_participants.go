package database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// JoinGatheringWithParticipants joins participants into a gathering. Returns the new number of participants
func JoinGatheringWithParticipants(manager *common_globals.MatchmakingManager, gatheringID uint32, connection *nex.PRUDPConnection, additionalParticipants []types.PID, joinMessage string, joinMatchmakeSessionBehavior constants.JoinMatchmakeSessionBehavior) (uint32, *nex.Error) {
	var ownerPID uint64
	var maxParticipants uint32
	var flags uint32
	var oldParticipants []uint64
	err := manager.Database.QueryRow(`SELECT owner_pid, max_participants, flags, participants FROM matchmaking.gatherings WHERE id=$1`, gatheringID).Scan(&ownerPID, &maxParticipants, &flags, pqextended.Array(&oldParticipants))
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if uint32(len(oldParticipants)+1+len(additionalParticipants)) > maxParticipants {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionFull, "change_error")
	}

	var newParticipants []uint64

	// * If joinMatchmakeSessionBehavior is set to 1, we check if the caller is already joined into the session
	if joinMatchmakeSessionBehavior == constants.JoinMatchmakeSessionBehaviorImAlreadyJoined {
		if !slices.Contains(oldParticipants, uint64(connection.PID())) {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "change_error")
		}
	} else {
		if slices.Contains(oldParticipants, uint64(connection.PID())) {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.AlreadyParticipatedGathering, "change_error")
		}

		// * Only include the caller as a new participant when they aren't joined
		newParticipants = []uint64{uint64(connection.PID())}
	}

	for _, participant := range additionalParticipants {
		newParticipants = append(newParticipants, uint64(participant))
	}

	participants := append(oldParticipants, newParticipants...)

	// * We have already checked that the gathering exists above, so we don't have to check the rows affected on sql.Result
	_, err = manager.Database.Exec(`UPDATE matchmaking.gatherings SET participants=$1 WHERE id=$2`, pqextended.Array(participants), gatheringID)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	// NOTE - This will log even if no new participants are added
	nexError := tracking.LogJoinGathering(manager.Database, connection.PID(), gatheringID, newParticipants, participants)
	if nexError != nil {
		return 0, nexError
	}

	var participantJoinedTargets []uint64

	// * When the VerboseParticipants or the VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
	if flags&(match_making.GatheringFlags.VerboseParticipants|match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
		participantJoinedTargets = common_globals.RemoveDuplicates(participants)
	} else {
		participantJoinedTargets = []uint64{ownerPID}
	}

	// * Send the switch SwitchGathering to the new participants first
	for _, participant := range common_globals.RemoveDuplicates(newParticipants) {
		// * Don't send the SwitchGathering notification to the participant that requested the join
		if uint64(connection.PID()) == participant {
			continue
		}

		notificationCategory := notifications.NotificationCategories.SwitchGathering
		notificationSubtype := notifications.NotificationSubTypes.SwitchGathering.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type = types.UInt32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
		oEvent.Param1 = types.UInt64(gatheringID)
		oEvent.Param2 = types.UInt64(participant)

		// * Send the notification to the participant
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{participant})
	}

	for _, participant := range newParticipants {
		// * If the new participant is the same as the owner, then we are creating a new gathering.
		// * We don't need to send the new participant notification event in that case
		if flags&(match_making.GatheringFlags.VerboseParticipants|match_making.GatheringFlags.VerboseParticipantsEx) != 0 || uint64(connection.PID()) != ownerPID {
			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = connection.PID()
			oEvent.Type = types.UInt32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
			oEvent.Param1 = types.UInt64(gatheringID)
			oEvent.Param2 = types.UInt64(participant)
			oEvent.StrParam = types.NewString(joinMessage)
			oEvent.Param3 = types.UInt64(len(participants))

			common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participantJoinedTargets)
		}

		// * This flag also sends a recap of all currently connected players on the gathering to the participant that is connecting
		if flags&match_making.GatheringFlags.VerboseParticipantsEx != 0 {
			// TODO - Should this actually be deduplicated?
			for _, oldParticipant := range common_globals.RemoveDuplicates(oldParticipants) {
				notificationCategory := notifications.NotificationCategories.Participation
				notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID()
				oEvent.Type = types.UInt32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
				oEvent.Param1 = types.UInt64(gatheringID)
				oEvent.Param2 = types.UInt64(oldParticipant)
				oEvent.StrParam = types.NewString(joinMessage)
				oEvent.Param3 = types.UInt64(len(participants))

				// * Send the notification to the joining participant
				common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{participant})
			}
		}
	}

	return uint32(len(participants)), nil
}
