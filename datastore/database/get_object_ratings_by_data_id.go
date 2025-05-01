package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetObjectRatingsByDataID(manager *common_globals.DataStoreManager, dataID types.UInt64) (types.List[datastore_types.DataStoreRatingInfoWithSlot], *nex.Error) {
	ratings := types.NewList[datastore_types.DataStoreRatingInfoWithSlot]()

	// TODO - If the total value goes above or below int64 min/max, it needs to be rounded to either the min/max
	rows, err := manager.Database.Query(`
		SELECT
			rs.slot,
			COALESCE(SUM(r.value), 0) + rs.initial_value AS total_value,
			COUNT(r.id) AS rating_count,
			rs.initial_value
		FROM
			datastore.rating_settings rs
		LEFT JOIN
			datastore.ratings r ON rs.data_id = r.data_id AND rs.slot = r.slot
		WHERE
			rs.data_id=$1
		GROUP BY
			rs.slot, rs.initial_value
		ORDER BY
			rs.slot
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
