package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func ObjectOwner(manager *common_globals.DataStoreManager, dataID types.UInt64) (types.PID, *nex.Error) {
	var owner types.PID

	err := manager.Database.QueryRow(`SELECT owner FROM datastore.objects WHERE data_id=$1`, dataID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return owner, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return owner, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return owner, nil
}
