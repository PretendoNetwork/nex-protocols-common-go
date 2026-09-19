package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// TODO - Add an optional flag to require DataFlagNeedCompletion being respected. Real servers seem to ignore it. See https://github.com/PretendoNetwork/swapdoodle/pull/1#issuecomment-4292881082 for details
func ObjectUploaded(database common_globals.Database, dataID types.UInt64) (bool, *nex.Error) {
	var uploaded bool

	err := database.QueryRow(`SELECT upload_completed FROM datastore.objects WHERE data_id=$1 AND deleted=false`, dataID).Scan(&uploaded)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return uploaded, nil
}
