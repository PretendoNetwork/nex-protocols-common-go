package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteObjectRatings(database common_globals.Database, dataID types.UInt64) *nex.Error {
	_, err := database.Exec(`
		WITH
			logs AS (
				DELETE FROM datastore.rating_logs WHERE data_id=$1
			),
			history AS (
				UPDATE datastore.rating_history SET removed = true WHERE data_id=$1 AND NOT removed
			)
		UPDATE datastore.rating_slots
		SET total_value = initial_value, rating_count = 0
		WHERE data_id=$1
	`, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// TODO - Do we care about result.RowsAffected() here?

	return nil
}
