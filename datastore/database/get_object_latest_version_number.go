package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetObjectLatestVersionNumber(database common_globals.Database, dataID types.UInt64) (uint32, *nex.Error) {
	var version uint32

	err := database.QueryRow(`SELECT MAX(version) FROM datastore.object_versions WHERE data_id=$1 AND completed = true GROUP BY data_id`, dataID).Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			return version, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return version, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return version, nil
}
