package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func CompleteObjectUpdate(database common_globals.Database, dataID types.UInt64, version types.UInt32) *nex.Error {
	_, err := database.Exec(`
		WITH completed AS (
			UPDATE datastore.object_versions
			SET completed = true
			WHERE data_id = $1 AND version = $2
			RETURNING data_id
		)
		UPDATE datastore.objects o
		SET
			update_date = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
			true_update_date = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
			expiration_date = CASE
				WHEN o.expiration_date = $3 THEN o.expiration_date
				ELSE (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') + (o.expiration_days * INTERVAL '1 day')
			END
		FROM completed c
		WHERE o.data_id = c.data_id AND NOT EXISTS (
			SELECT 1
			FROM datastore.object_versions v
			WHERE v.data_id = $1 AND v.version > $2 AND v.completed = true
		)
	`, dataID, version, types.NewPermanentDateTime())

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
