package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetObjectLatestVersionNumber(manager *common_globals.DataStoreManager, dataID types.UInt64) (uint16, *nex.Error) {
	var version uint16

	err := manager.Database.QueryRow(`SELECT version FROM datastore.objects WHERE data_id=$1`, dataID).Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			return version, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return version, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return version, nil
}
