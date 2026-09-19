package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func UnperpetuateObjectByDataID(database common_globals.Database, dataID uint64, deleteObject types.Bool) *nex.Error {
	var err error

	if deleteObject {
		_, err = database.Exec(`UPDATE datastore.objects SET deleted = TRUE WHERE data_id=$1`, dataID)
	} else {
		var expirationDays uint8
		err = database.QueryRow(`SELECT expiration_days FROM datastore.objects WHERE data_id=$1`, dataID).Scan(&expirationDays)
		if err != nil {
			// TODO - Send more specific errors?
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		expirationDate := time.Now().UTC().Add(time.Duration(expirationDays) * 24 * time.Hour)

		_, err = database.Exec(`UPDATE datastore.objects SET expiration_date = $1 WHERE data_id=$2`, expirationDate, dataID)
	}

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
