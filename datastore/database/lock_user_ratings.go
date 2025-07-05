package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func LockUserRatings(manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, settings datastore_types.DataStoreRatingInitParam) *nex.Error {
	expirationDate := manager.CalculateRatingExpirationTime(*manager, settings)

	// TODO - Check rows affected?
	_, err := manager.Database.Exec(`
		INSERT INTO datastore.rating_locks (
			pid,
			data_id,
			slot,
			locked_until
		) VALUES (
			$1,
			$2,
			$3,
			$4
		)
		ON CONFLICT (pid, data_id, slot)
			DO UPDATE
				SET locked_until = $4
	`, pid, dataID, slot, expirationDate)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
