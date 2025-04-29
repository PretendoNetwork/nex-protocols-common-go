package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetGetMetaObjectInfoByPersistenceTarget(manager *common_globals.DataStoreManager, persistenceTarget datastore_types.DataStorePersistenceTarget) (datastore_types.DataStoreMetaInfo, types.UInt64, *nex.Error) {
	var dataID types.UInt64
	var metaInfo datastore_types.DataStoreMetaInfo // * Only used for error responses
	var accessPassword types.UInt64                // * Only used for error responses

	err := manager.Database.QueryRow(
		`SELECT data_id FROM datastore.persistence_slots WHERE pid=$1 AND slot=$2`,
		persistenceTarget.OwnerID, persistenceTarget.PersistenceSlotID,
	).Scan(&dataID)

	if err != nil {
		if err == sql.ErrNoRows {
			return metaInfo, accessPassword, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return metaInfo, accessPassword, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return GetGetMetaObjectInfoByDataID(manager, dataID)
}
