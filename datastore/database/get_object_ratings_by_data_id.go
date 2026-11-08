package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetObjectRatingsByDataID(database common_globals.Database, dataID types.UInt64) (types.List[datastore_types.DataStoreRatingInfoWithSlot], *nex.Error) {
	ratings := types.NewList[datastore_types.DataStoreRatingInfoWithSlot]()

	rows, err := database.Query(`
		SELECT
			slot,
			total_value,
			rating_count,
			initial_value
		FROM datastore.rating_slots
		WHERE data_id=$1
		ORDER BY slot
		`, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		rating := datastore_types.NewDataStoreRatingInfoWithSlot()

		err := rows.Scan(&rating.Slot, &rating.Rating.TotalValue, &rating.Rating.Count, &rating.Rating.InitialValue)
		if err != nil {
			// TODO - Send more specific errors?
			return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		ratings = append(ratings, rating)
	}

	if err := rows.Err(); err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return ratings, nil
}
