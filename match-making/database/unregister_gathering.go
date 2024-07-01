package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
)

// UnregisterGathering unregisters a given gathering on a database
func UnregisterGathering(db *sql.DB, id uint32) *nex.Error {
	result, err := db.Exec(`UPDATE matchmaking.gatherings SET registered=false WHERE id=$1`, id)
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
