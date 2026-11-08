package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func GetPerpetuatedObjectID(database common_globals.Database, ownerPID types.PID, slot types.UInt16) (uint64, *nex.Error) {
	var dataID uint64

	err := database.QueryRow(`
		SELECT s.data_id
		FROM datastore.persistence_slots s
		JOIN datastore.objects o ON o.data_id = s.data_id
		WHERE s.pid=$1 AND s.slot=$2 AND o.deleted=false
	`, ownerPID, slot).Scan(&dataID)

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
