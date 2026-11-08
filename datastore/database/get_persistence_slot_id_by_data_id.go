package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func GetPersistenceSlotIDByDataID(database common_globals.Database, dataID types.UInt64) (uint16, *nex.Error) {
	var slot uint16

	err := database.QueryRow(`SELECT slot FROM datastore.persistence_slots WHERE data_id=$1`, dataID).Scan(&slot)

	if err != nil {
		if err == sql.ErrNoRows {
			// * The object is not in a slot
			return datastore_constants.InvalidPersistenceSlotID, nil
		}

		// TODO - Send more specific errors?
		return datastore_constants.InvalidPersistenceSlotID, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return slot, nil
}
