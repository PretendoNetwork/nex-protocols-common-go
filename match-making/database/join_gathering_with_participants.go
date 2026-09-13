package database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_constants "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	notifications_constants "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/constants"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

func SendAddedToGatheringToNewParticipants(connection *nex.PRUDPConnection, newParticipants []uint64, gatheringID uint32) {
	for _, participant := range common_globals.RemoveDuplicates(newParticipants) {
		// * Don't send the SwitchGathering notification to the participant that requested the join
		if uint64(connection.PID()) == participant {
			continue
		}

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type = notifications_constants.NotificationCategoryAddedToGathering.Build()
		oEvent.Param1 = types.UInt64(gatheringID)
		oEvent.Param2 = types.UInt64(participant)

		// * Send the notification to the participant
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{participant})
	}
}

func SendJoinNotificationsOfTo(connection *nex.PRUDPConnection, gatheringID uint32, joinMessage string, participantCount int, sourceParticipants []uint64, destinationParticipants []uint64) {
	for _, participant := range common_globals.RemoveDuplicates(sourceParticipants) {
		destWithoutSelf := common_globals.RemoveDuplicates(destinationParticipants)

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type = notifications_constants.NotificationCategoryParticipationEvent.Build(notifications_constants.ParticipationEventsParticipate)
		oEvent.Param1 = types.UInt64(gatheringID)
		oEvent.Param2 = types.UInt64(participant)
		oEvent.StrParam = types.NewString(joinMessage)
		oEvent.Param3 = types.UInt64(uint64(participantCount))

		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, destinationParticipants)
	}
}

// JoinGatheringWithParticipants joins participants into a gathering. Returns the new number of participants
func JoinGatheringWithParticipants(manager *common_globals.MatchmakingManager, gatheringID uint32, connection *nex.PRUDPConnection, additionalParticipants []types.PID, joinMessage string, joinMatchmakeSessionBehavior constants.JoinMatchmakeSessionBehavior) (uint32, *nex.Error) {
	var ownerPID uint64
	var maxParticipants uint32
	var flags match_making_constants.GatheringFlags
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

	// * Send the switch SwitchGathering to the new participants first
	SendAddedToGatheringToNewParticipants(connection, newParticipants, gatheringID)
	if flags.HasFlag(match_making_constants.GatheringFlagNotifyParticipationEventsToAllParticipants) || flags.HasFlag(match_making_constants.GatheringFlagNotifyParticipationEventsToAllParticipantsReproducibly) {
		// inform all people of the new participants joining
		SendJoinNotificationsOfTo(connection, gatheringID, joinMessage, len(participants), newParticipants, oldParticipants)
	} else {
		// inform the owner of all people who are joining
		SendJoinNotificationsOfTo(connection, gatheringID, joinMessage, len(participants), newParticipants, []uint64{ownerPID})
	}

	if flags.HasFlag(match_making_constants.GatheringFlagNotifyParticipationEventsToAllParticipantsReproducibly) {
		// send the join notifications to all the new players for every player such that the new players all know of every joined player
		SendJoinNotificationsOfTo(connection, gatheringID, joinMessage, len(participants), oldParticipants, newParticipants)
	}

	return uint32(len(participants)), nil
}
