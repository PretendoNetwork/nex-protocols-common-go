package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func GetPerpetuatedObject(manager *common_globals.DataStoreManager, ownerPID types.PID, slot types.UInt16) (uint64, bool, *nex.Error) {
	var dataID uint64
	var deleteObject bool

	err := manager.Database.QueryRow(`
		SELECT data_id, delete_last_object
		FROM datastore.persistence_slots
		WHERE pid=$1 AND slot=$2`, ownerPID, slot).Scan(&dataID, &deleteObject)

	if err != nil {
		if err == sql.ErrNoRows {
			// * No object in the slot
			return datastore_constants.InvalidDataID, false, nil
		}

		// TODO - Send more specific errors?
		return datastore_constants.InvalidDataID, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return dataID, deleteObject, nil
}
