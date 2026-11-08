package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetObjectUpdatePreparer(database common_globals.Database, dataID types.UInt64, version types.UInt32) (types.PID, *nex.Error) {
	var preparer types.PID

	err := database.QueryRow(`
		SELECT v.preparer
		FROM datastore.object_versions v
		JOIN datastore.objects o ON o.data_id = v.data_id
		WHERE v.data_id = $1 AND v.version = $2 AND v.version > 1 AND o.deleted = false
	`, dataID, version).Scan(&preparer)

	if err != nil {
		if err == sql.ErrNoRows {
			return preparer, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return preparer, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return preparer, nil
}
