package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
)

// UpdateParticipation updates the participation of a matchmake session
func UpdateParticipation(db *sql.DB, gatheringID uint32, participation bool) *nex.Error {
	result, err := db.Exec(`UPDATE matchmaking.matchmake_sessions SET open_participation=$1 WHERE id=$2`, participation, gatheringID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	if rowsAffected == 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	return nil
}