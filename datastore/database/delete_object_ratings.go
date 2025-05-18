package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func DeleteObjectRatings(manager *common_globals.DataStoreManager, dataID types.UInt64) *nex.Error {
	// TODO - Should we just LOGICALLY delete these, like we do everything else, or continue to HARD delete them?
	_, err := manager.Database.Exec(`DELETE FROM datastore.ratings WHERE data_id=$1`, dataID)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// TODO - Do we care about result.RowsAffected() here?

	return nil
}
