package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func AddUserRating(database common_globals.Database, dataID types.UInt64, slot types.UInt8, pid types.PID, ratingValue types.Int32) *nex.Error {
	// TODO - Check rows affected?
	_, err := database.Exec(`
		INSERT INTO datastore.ratings (
			data_id,
			slot,
			pid,
			value,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			(CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
			(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
		)
	`, ratingValue, dataID, slot, pid)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
