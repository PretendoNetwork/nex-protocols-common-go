package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

// * GetSpecificObjectInfoByDataID is only used by GetSpecificMeta(V1).
// * Has a dedicated function because those methods ignore things like
// * deletion state, access permissions, expiration, etc.
func GetSpecificObjectInfoByDataID(database common_globals.Database, dataID types.UInt64) (datastore_types.DataStoreMetaInfo, uint32, bool, *nex.Error) {
	var metaInfo datastore_types.DataStoreMetaInfo
	var version uint32
	var metaBinaryIsNull bool // * Currently the only way to check if the MetaBinary is empty when posted or not

	err := database.QueryRow(`
		SELECT
			data_id,
			owner,
			size,
			data_type,
			meta_binary,
			meta_binary IS NULL,
			raw_flags,
			version
		FROM datastore.objects
		WHERE data_id=$1`, dataID).Scan(
		&metaInfo.DataID,
		&metaInfo.OwnerID,
		&metaInfo.Size,
		&metaInfo.DataType,
		&metaInfo.MetaBinary,
		&metaBinaryIsNull,
		&metaInfo.Flag,
		&version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return metaInfo, version, metaBinaryIsNull, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return metaInfo, version, metaBinaryIsNull, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return metaInfo, version, metaBinaryIsNull, nil
}
