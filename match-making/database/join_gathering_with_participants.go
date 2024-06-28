package database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// JoinGatheringWithParticipants joins participants into a gathering. Returns the new number of participants
func JoinGatheringWithParticipants(db *sql.DB, gatheringID uint32, connection *nex.PRUDPConnection, additionalParticipants []*types.PID, joinMessage string) (uint32, *nex.Error) {
	var ownerPID uint64
	var maxParticipants uint32
	var flags uint32
	var oldParticipants []uint64
	err := db.QueryRow(`SELECT owner_pid, max_participants, flags, participants FROM matchmaking.gatherings WHERE id=$1`, gatheringID).Scan(&ownerPID, &maxParticipants, &flags, pqextended.Array(&oldParticipants))
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if uint32(len(oldParticipants) + 1 + len(additionalParticipants)) > maxParticipants {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionFull, "change_error")
	}

	if slices.Contains(oldParticipants, connection.PID().Value()) {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.AlreadyParticipatedGathering, "change_error")
	}

	newParticipants := []uint64{connection.PID().Value()}
	for _, participant := range additionalParticipants {
		newParticipants = append(newParticipants, participant.Value())
	}

	participants := append(oldParticipants, newParticipants...)

	// * We have already checked that the gathering exists above, so we don't have to check the rows affected on sql.Result
	_, err = db.Exec(`UPDATE matchmaking.gatherings SET participants=$1 WHERE id=$2`, pqextended.Array(participants), gatheringID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	var participantJoinedTargets []uint64

	// * When the VerboseParticipants or the VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
	if flags & (match_making.GatheringFlags.VerboseParticipants | match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
		participantJoinedTargets = participants
	} else {
		participantJoinedTargets = []uint64{ownerPID}
	}

	for _, participant := range newParticipants {
		// * If the new participant is the same as the owner, then we are creating a new gathering.
		// * We don't need to send the new participant notification event in that case
		if flags & (match_making.GatheringFlags.VerboseParticipants | match_making.GatheringFlags.VerboseParticipantsEx) != 0 || connection.PID().Value() != ownerPID {
			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = connection.PID()
			oEvent.Type.Value = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
			oEvent.Param1.Value = gatheringID
			oEvent.Param2.Value = uint32(participant) // TODO - This assumes a legacy client. Will not work on the Switch
			oEvent.StrParam.Value = joinMessage
			oEvent.Param3.Value = uint32(len(participants))

			common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participantJoinedTargets)
		}

		// * This flag also sends a recap of all currently connected players on the gathering to the participant that is connecting
		if flags & match_making.GatheringFlags.VerboseParticipantsEx != 0 {
			for _, oldParticipant := range oldParticipants {
				notificationCategory := notifications.NotificationCategories.Participation
				notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

				oEvent := notifications_types.NewNotificationEvent()
				oEvent.PIDSource = connection.PID()
				oEvent.Type.Value = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
				oEvent.Param1.Value = gatheringID
				oEvent.Param2.Value = uint32(oldParticipant) // TODO - This assumes a legacy client. Will not work on the Switch
				oEvent.StrParam.Value = joinMessage
				oEvent.Param3.Value = uint32(len(participants))

				// * Send the notification to the joining participant
				common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{participant})
			}
		}

		// * Don't send the SwitchGathering notification to the participant that requested the join
		if connection.PID().Value() == uint64(participant) {
			continue
		}

		notificationCategory := notifications.NotificationCategories.SwitchGathering
		notificationSubtype := notifications.NotificationSubTypes.SwitchGathering.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type.Value = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
		oEvent.Param1.Value = gatheringID
		oEvent.Param2.Value = uint32(participant) // TODO - This assumes a legacy client. Will not work on the Switch

		// * Send the notification to the participant
		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{participant})
	}

	return uint32(len(participants)), nil
}
