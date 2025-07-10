package database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// JoinGathering joins participants from the same connection into a gathering. Returns the new number of participants
func JoinGathering(manager *common_globals.MatchmakingManager, gatheringID uint32, connection *nex.PRUDPConnection, vacantParticipants uint16, joinMessage string) (uint32, *nex.Error) {
	// * vacantParticipants represents the total number of participants that are joining (including the main participant)
	// * Prevent underflow below if vacantParticipants is set to zero
	if vacantParticipants == 0 {
		vacantParticipants = 1
	}

	var ownerPID uint64
	var maxParticipants uint32
	var flags uint32
	var participants []uint64
	err := manager.Database.QueryRow(`SELECT owner_pid, max_participants, flags, participants FROM matchmaking.gatherings WHERE id=$1`, gatheringID).Scan(&ownerPID, &maxParticipants, &flags, pqextended.Array(&participants))
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if maxParticipants != 0 {
		if uint32(len(participants))+uint32(vacantParticipants) > maxParticipants {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionFull, "change_error")
		}
	}

	if slices.Contains(participants, uint64(connection.PID())) {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.AlreadyParticipatedGathering, "change_error")
	}

	var newParticipants []uint64

	// * Additional participants are represented by duplicating the main participant PID on the array
	for range vacantParticipants {
		newParticipants = append(newParticipants, uint64(connection.PID()))
	}

	var totalParticipants []uint64 = append(newParticipants, participants...)

	// * We have already checked that the gathering exists above, so we don't have to check the rows affected on sql.Result
	_, err = manager.Database.Exec(`UPDATE matchmaking.gatherings SET participants=$1 WHERE id=$2`, pqextended.Array(totalParticipants), gatheringID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	nexError := tracking.LogJoinGathering(manager.Database, connection.PID(), gatheringID, newParticipants, totalParticipants)
	if nexError != nil {
		return 0, nexError
	}

	var participantJoinedTargets []uint64

	// * When the VerboseParticipants or VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
	if flags&(match_making.GatheringFlags.VerboseParticipants|match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
		participantJoinedTargets = common_globals.RemoveDuplicates(totalParticipants)
	} else {
		// * If the new participant is the same as the owner, then we are creating a new gathering.
		// * We don't need to send notification events in that case
		if uint64(connection.PID()) == ownerPID {
			return uint32(len(totalParticipants)), nil
		}

		participantJoinedTargets = []uint64{ownerPID}
	}

	notificationCategory := notifications.NotificationCategories.Participation
	notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID()
	oEvent.Type = types.UInt32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
	oEvent.Param1 = types.UInt64(gatheringID)
	oEvent.Param2 = types.UInt64(connection.PID())
	oEvent.StrParam = types.NewString(joinMessage)
	oEvent.Param3 = types.UInt64(len(totalParticipants))

	common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participantJoinedTargets)

	// * This flag also sends a recap of all currently connected players on the gathering to the participant that is connecting
	if flags&match_making.GatheringFlags.VerboseParticipantsEx != 0 {
		// TODO - Should this actually be deduplicated?
		for _, participant := range common_globals.RemoveDuplicates(participants) {
			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = connection.PID()
			oEvent.Type = types.UInt32(notifications.BuildNotificationType(notificationCategory, notificationSubtype))
			oEvent.Param1 = types.UInt64(gatheringID)
			oEvent.Param2 = types.UInt64(participant)
			oEvent.StrParam = types.NewString(joinMessage)
			oEvent.Param3 = types.UInt64(len(totalParticipants))

			// * Send the notification to the joining participant
			common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{uint64(connection.PID())})
		}
	}

	return uint32(len(totalParticipants)), nil
}
