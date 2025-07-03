package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetObjectRatingByDataIDAndSlot(manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8) (datastore_types.DataStoreRatingInfo, *nex.Error) {
	ratingInfo := datastore_types.NewDataStoreRatingInfo()

	// TODO - If the total value goes above or below int64 min/max, it needs to be rounded to either the min/max
	err := manager.Database.QueryRow(`
		SELECT
			COALESCE(SUM(r.value), 0) + rs.initial_value AS total_value,
			COUNT(r.id) AS rating_count,
			rs.initial_value
		FROM
			datastore.rating_settings AS rs
		LEFT JOIN
			datastore.ratings r ON rs.data_id = r.data_id AND rs.slot = r.slot
		WHERE
			rs.data_id=$1 AND rs.slot=$2
		GROUP BY
			rs.initial_value
		`, dataID, slot).Scan(&ratingInfo.TotalValue, &ratingInfo.Count, &ratingInfo.InitialValue)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return ratingInfo, nex.NewError(nex.ResultCodes.DataStore.NotFound, "Rating slot not found")
		}

		// TODO - Send more specific errors?
		return ratingInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return ratingInfo, nil
}
