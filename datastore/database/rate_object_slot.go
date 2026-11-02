package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
)

func RateObjectSlot(database common_globals.Database, dataID types.UInt64, slot types.UInt8, pid types.PID, value types.Int32) *nex.Error {
	_, err := database.Exec(`
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

	return nil
}
