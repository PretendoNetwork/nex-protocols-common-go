package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func GetPerpetuatedObjectID(manager *common_globals.DataStoreManager, ownerPID types.PID, slot types.UInt16) (uint64, *nex.Error) {
	var dataID uint64

	err := manager.Database.QueryRow(`SELECT data_id FROM datastore.persistence_slots WHERE pid=$1 AND slot=$2`, ownerPID, slot).Scan(&dataID)

	if err != nil {
		if err == sql.ErrNoRows {
			// * No object in the slot
			return datastore_constants.InvalidDataID, nil
		}

		// TODO - Send more specific errors?
		return datastore_constants.InvalidDataID, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return dataID, nil
}
