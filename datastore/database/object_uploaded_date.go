package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func ObjectCreationDate(manager *common_globals.DataStoreManager, dataID types.UInt64) (time.Time, *nex.Error) {
	var creationDate time.Time

	err := manager.Database.QueryRow(`SELECT creation_date FROM datastore.objects WHERE data_id=$1`, dataID).Scan(&creationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return creationDate, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return creationDate, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return creationDate, nil
}
