package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func InsertObjectUpdateByPrepareUpdateParam(transaction common_globals.Database, preparer types.PID, updateParam datastore_types.DataStorePrepareUpdateParam, currentData datastore_types.DataStoreMetaInfo) (types.UInt32, *nex.Error) {
	// * Unlike when creating an object, updating it has no size limit for some reason
	_, err := transaction.Exec(`SELECT 1 FROM datastore.objects WHERE data_id = $1 FOR UPDATE`, currentData.DataID)
	if err != nil {
		// TODO - Send more specific errors?
		return 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	var version types.UInt32

	// * The next version is always sequential even if older versions aren't
	// * completed yet
	err = transaction.QueryRow(`
		INSERT INTO datastore.object_versions (
			data_id,
			version,
			size,
			extra_data,
			preparer
		)
		SELECT
			data_id,
			MAX(version) + 1,
			$2,
			$3,
			$4
		FROM datastore.object_versions
		WHERE data_id = $1
		GROUP BY data_id
		RETURNING version
	`, currentData.DataID, updateParam.Size, pq.Array(updateParam.ExtraData), preparer).Scan(&version)

	if err != nil {
		// TODO - Send more specific errors?
		return 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return version, nil
}
