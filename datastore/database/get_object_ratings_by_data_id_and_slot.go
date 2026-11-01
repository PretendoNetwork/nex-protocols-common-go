package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetObjectRatingByDataIDAndSlot(database common_globals.Database, dataID types.UInt64, slot types.UInt8) (datastore_types.DataStoreRatingInfo, *nex.Error) {
	ratingInfo := datastore_types.NewDataStoreRatingInfo()

	err := database.QueryRow(`
		SELECT
			total_value,
			rating_count,
			initial_value
		FROM datastore.rating_slots
		WHERE data_id=$1 AND slot=$2
		`, dataID, slot).Scan(&ratingInfo.TotalValue, &ratingInfo.Count, &ratingInfo.InitialValue)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return ratingInfo, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Rating slot not found")
		}

		// TODO - Send more specific errors?
		return ratingInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return ratingInfo, nil
}
