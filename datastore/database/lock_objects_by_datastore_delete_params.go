package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func LockObjectsByDataStoreDeleteParams(database common_globals.Database, params types.List[datastore_types.DataStoreDeleteParam]) *nex.Error {
	dataIDs := make([]types.UInt64, 0, len(params))

	for _, param := range params {
		dataIDs = append(dataIDs, param.DataID)
	}

	_, err := database.Exec(`
		SELECT 1
		FROM datastore.objects
		WHERE data_id = ANY($1::numeric[])
		ORDER BY data_id
		FOR UPDATE
	`, pq.Array(dataIDs))
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
