package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func RateObjectSlot(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, value types.Int32, settings datastore_types.DataStoreRatingInitParam) *nex.Error {
	// * The lock created by rateObjectSlot only exists until the end of the
	// * transaction, so if this isn't a transaction then we need to start one
	db, ok := database.(*sql.DB)
	if !ok {
		return rateObjectSlot(database, manager, dataID, slot, pid, value, settings)
	}

	transaction, err := db.Begin()
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Does nothing once the transaction has been committed successfully
	defer transaction.Rollback()

	errCode := rateObjectSlot(transaction, manager, dataID, slot, pid, value, settings)
	if errCode != nil {
		return errCode
	}

	if err := transaction.Commit(); err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}

func rateObjectSlot(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, value types.Int32, settings datastore_types.DataStoreRatingInitParam) *nex.Error {
	// * Lock until the transaction ends
	_, err := database.Exec(`
		SELECT 1
		FROM datastore.rating_slots
		WHERE data_id = $1 AND slot = $2
		FOR UPDATE
	`, dataID, slot)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Check this after the slot is locked otherwise multiple ratings from
	// * the same user at one time could both get past the lock
	if settings.LockType != datastore_constants.RatingLockNone {
		ratingLog, errCode := GetUserRatingLog(database, dataID, slot, pid)
		if errCode != nil {
			return errCode
		}

		if !ratingLog.LockExpirationTime.IsNever() && time.Now().UTC().Before(ratingLog.LockExpirationTime.Standard()) {
			return nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
		}
	}

	_, err = database.Exec(`
		WITH
			previous AS (
				SELECT h.value
				FROM datastore.rating_history h
				JOIN datastore.rating_slots rs ON rs.data_id = h.data_id AND rs.slot = h.slot
				WHERE h.data_id = $1 AND h.slot = $2 AND h.pid = $3 AND NOT h.removed AND rs.allow_multiple_ratings
				ORDER BY h.id DESC
				LIMIT 1
			),
			rated_slot AS (
				UPDATE datastore.rating_slots rs
				SET
					total_value = CASE
						WHEN rs.round_negatives THEN GREATEST(rs.total_value - COALESCE((SELECT value FROM previous), 0) + $4, 0)
						ELSE rs.total_value - COALESCE((SELECT value FROM previous), 0) + $4
					END,
					rating_count = rs.rating_count + CASE WHEN EXISTS (SELECT 1 FROM previous) THEN 0 ELSE 1 END
				WHERE rs.data_id = $1 AND rs.slot = $2
				RETURNING rs.data_id, rs.slot
			),
			history AS (
				INSERT INTO datastore.rating_history (
					data_id,
					slot,
					pid,
					value
				)
				SELECT data_id, slot, $3, $4 FROM rated_slot
			)
		INSERT INTO datastore.rating_logs (
			data_id,
			slot,
			pid,
			value
		)
		SELECT data_id, slot, $3, $4 FROM rated_slot
		ON CONFLICT (data_id, slot, pid) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
	`, dataID, slot, pid, value)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "check_violation" {
			return nex.NewError(nex.ResultCodes.PythonCore.Exception, "Rating would overflow past max int64")
		}

		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if settings.LockType != datastore_constants.RatingLockNone {
		return LockUserRatings(database, manager, dataID, slot, pid, settings)
	}

	return nil
}
