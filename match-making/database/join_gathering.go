package database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// JoinGathering joins participants from the same connection into a gathering. Returns the new number of participants
func JoinGathering(db *sql.DB, gatheringID uint32, connection *nex.PRUDPConnection, vacantParticipants uint16, joinMessage string) (uint32, *nex.Error) {
	// * vacantParticipants represents the total number of participants that are joining (including the main participant)
	// * Prevent underflow below if vacantParticipants is set to zero
	if vacantParticipants == 0 {
		vacantParticipants = 1
	}

	var ownerPID uint64
	var maxParticipants uint32
	var flags uint32
	var participants []uint64
	err := db.QueryRow(`SELECT owner_pid, max_participants, flags, participants FROM matchmaking.gatherings WHERE id=$1`, gatheringID).Scan(&ownerPID, &maxParticipants, &flags, pqextended.Array(&participants))
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	if uint32(len(participants)) + uint32(vacantParticipants) > maxParticipants {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionFull, "change_error")
	}

	if slices.Contains(participants, connection.PID().Value()) {
		return 0, nex.NewError(nex.ResultCodes.RendezVous.AlreadyParticipatedGathering, "change_error")
	}

	// TODO - This won't work on the Switch!
	newParticipants := append(participants, connection.PID().Value())

	// * Additional participants are represented as the negative of the main participant PID
	// * These are casted to an unsigned value for compatibility with uint32 and uint64
	additionalParticipant := int32(-connection.PID().LegacyValue())
	for range vacantParticipants - 1 {
		newParticipants = append(newParticipants, uint64(uint32(additionalParticipant)))
	}

	// * We have already checked that the gathering exists above, so we don't have to check the rows affected on sql.Result
	_, err = db.Exec(`UPDATE matchmaking.gatherings SET participants=$1 WHERE id=$2`, pqextended.Array(newParticipants), gatheringID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	var participatJoinedTargets []uint64

	// * When the VerboseParticipants or VerboseParticipantsEx flags are set, all participant notification events are sent to everyone
	if flags & (match_making.GatheringFlags.VerboseParticipants | match_making.GatheringFlags.VerboseParticipantsEx) != 0 {
		participatJoinedTargets = newParticipants
	} else {
		// * If the new participant is the same as the owner, then we are creating a new gathering.
		// * We don't need to send notification events in that case
		if connection.PID().Value() == ownerPID {
			return uint32(len(newParticipants)), nil
		}

		participatJoinedTargets = []uint64{ownerPID}
	}

	notificationCategory := notifications.NotificationCategories.Participation
	notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID()
	oEvent.Type.Value = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
	oEvent.Param1.Value = gatheringID
	oEvent.Param2.Value = uint32(connection.PID().LegacyValue()) // TODO - This assumes a legacy client. Will not work on the Switch
	oEvent.StrParam.Value = joinMessage
	oEvent.Param3.Value = uint32(len(newParticipants))

	common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, participatJoinedTargets)

	// * This flag also sends a recap of all currently connected players on the gathering to the participant that is connecting
	if flags & match_making.GatheringFlags.VerboseParticipantsEx != 0 {
		for _, participant := range participants {
			notificationCategory := notifications.NotificationCategories.Participation
			notificationSubtype := notifications.NotificationSubTypes.Participation.NewParticipant

			oEvent := notifications_types.NewNotificationEvent()
			oEvent.PIDSource = connection.PID()
			oEvent.Type.Value = notifications.BuildNotificationType(notificationCategory, notificationSubtype)
			oEvent.Param1.Value = gatheringID
			oEvent.Param2.Value = uint32(participant) // TODO - This assumes a legacy client. Will not work on the Switch
			oEvent.StrParam.Value = joinMessage
			oEvent.Param3.Value = uint32(len(newParticipants))

			// * Send the notification to the joining participant
			common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, []uint64{connection.PID().Value()})
		}
	}

	return uint32(len(newParticipants)), nil
}
