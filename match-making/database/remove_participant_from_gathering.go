package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// RemoveParticipantFromGathering removes a participant from a gathering. Returns the new list of participants
func RemoveParticipantFromGathering(db *sql.DB, gatheringID uint32, participant uint64) ([]uint64, *nex.Error) {
	var newParticipants []uint64
	err := db.QueryRow(`UPDATE matchmaking.gatherings SET participants=array_remove(participants, $1) WHERE id=$2 RETURNING participants`, participant, gatheringID).Scan(pqextended.Array(&newParticipants))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return newParticipants, nil
}
