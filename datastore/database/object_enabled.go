package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func ObjectEnabled(manager *common_globals.DataStoreManager, dataID types.UInt64) (bool, *nex.Error) {
	var uploaded bool
	var status types.UInt8

	err := manager.Database.QueryRow(`SELECT upload_completed, status FROM datastore.objects WHERE data_id=$1`, dataID).Scan(&uploaded, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * If either flag is set, assume enabled already
	enabled := uploaded || status == types.UInt8(datastore_constants.DataStatusNone)

	return enabled, nil
}
