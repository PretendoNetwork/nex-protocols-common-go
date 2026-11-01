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
				SELECT l.value
				FROM datastore.rating_logs l
				JOIN datastore.rating_slots rs ON rs.data_id = l.data_id AND rs.slot = l.slot
				WHERE l.data_id = $1 AND l.slot = $2 AND l.pid = $3 AND rs.allow_multiple_ratings
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
