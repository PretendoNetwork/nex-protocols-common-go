package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func EnableObject(manager *common_globals.DataStoreManager, dataID types.UInt64) *nex.Error {
	_, err := manager.Database.Exec(`UPDATE datastore.objects SET status = 0, upload_completed = TRUE WHERE data_id=$1`, dataID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
