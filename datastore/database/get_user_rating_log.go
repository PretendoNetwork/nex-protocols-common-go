package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetUserRatingLog(manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID) (datastore_types.DataStoreRatingLog, *nex.Error) {
	var ratingLog datastore_types.DataStoreRatingLog
	var ratingValue sql.NullInt64
	var lockedUntil sql.NullTime

	ratingLog.PID = pid
	ratingLog.RatingValue = -1 // * Default value for when a rating does not exist

	err := manager.Database.QueryRow(`
		SELECT
			r.value,
			l.locked_until
		FROM
			(
				SELECT
					value,
					updated_at
				FROM datastore.ratings
				WHERE data_id = $1 AND slot = $2 AND pid = $3
				ORDER BY updated_at DESC
				LIMIT 1
			) r
		LEFT JOIN datastore.rating_locks l ON
			l.data_id = $1 AND l.slot = $2 AND l.pid = $3
	`, dataID, slot, pid).Scan(&ratingValue, &lockedUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			// * This is valid, not everyone has ratings stored
			return ratingLog, nil
		}

		// TODO - Send more specific errors?
		return ratingLog, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if ratingValue.Valid {
		ratingLog.IsRated = true
		ratingLog.RatingValue = types.NewInt32(int32(ratingValue.Int64))

		if lockedUntil.Valid {
			ratingLog.LockExpirationTime.FromTimestamp(lockedUntil.Time.UTC())
		}
	}

	return ratingLog, nil
}
