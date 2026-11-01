package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteObjectRatingsBySlot(database common_globals.Database, dataID types.UInt64, slot types.UInt8) *nex.Error {
	result, err := database.Exec(`
		WITH logs AS (
			DELETE FROM datastore.rating_logs WHERE data_id=$1 AND slot=$2
		)
		UPDATE datastore.rating_slots
		SET total_value = initial_value, rating_count = 0
		WHERE data_id=$1 AND slot=$2
	`, dataID, slot)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	affected, err := result.RowsAffected()
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if affected == 0 {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Rating slot not found")
	}

	return nil
}
