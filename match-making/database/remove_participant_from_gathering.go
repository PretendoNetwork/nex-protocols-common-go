package database

import (
	"database/sql"
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// RemoveParticipantFromGathering removes a participant from a gathering. Returns the new list of participants
func RemoveParticipantFromGathering(db *sql.DB, gatheringID uint32, participant uint64) ([]uint64, *nex.Error) {
	newParticipantsStatement := fmt.Sprintf("array_remove(participants, %d)", participant)

	// * If the participant fits within a int32, then it is compatible with the 3DS and Wii U.
	// * Thus, we have to remove the additional participants too
	if uint64(int32(participant)) == participant {
		additionalParticipants := int32(participant)
		newParticipantsStatement = fmt.Sprintf("array_remove(array_remove(participants, %d), %d)", -additionalParticipants, participant)
	}

	var newParticipants []uint64
	err := db.QueryRow(`UPDATE matchmaking.gatherings SET participants=` + newParticipantsStatement + ` WHERE id=$1 RETURNING participants`, gatheringID).Scan(pqextended.Array(&newParticipants))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return newParticipants, nil
}
