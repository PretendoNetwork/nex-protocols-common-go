package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func LockUserRatings(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8, pid types.PID, settings datastore_types.DataStoreRatingInitParam) *nex.Error {
	expirationDate := manager.CalculateRatingExpirationTime(settings)

	// TODO - Check rows affected?
	_, err := database.Exec(`
		UPDATE datastore.rating_logs
		SET locked_until = $4
		WHERE pid = $1 AND data_id = $2 AND slot = $3
	`, pid, dataID, slot, expirationDate)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
