package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func UpdateUserRating(manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, ratingValue types.Int32) *nex.Error {
	// TODO - Check rows affected?
	_, err := manager.Database.Exec(`
		INSERT INTO datastore.ratings (
			data_id,
			slot,
			pid,
			value,
			created_at,
			updated_at
		) VALUES (
			$1,
			$2,
			$3,
			$4,
			(CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
			(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
		)
		ON CONFLICT (data_id, slot, pid)
			DO UPDATE
				SET value = $4,
					updated_at = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
	`, dataID, slot, pid, ratingValue)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
