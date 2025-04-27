package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetChangeMetaObjectInfoByDataID(manager *common_globals.DataStoreManager, dataID types.UInt64) (datastore_types.DataStoreMetaInfo, types.UInt64, *nex.Error) {
	var metaInfo datastore_types.DataStoreMetaInfo
	var creationDate time.Time
	var updateDate time.Time
	var lastReferenceDate time.Time
	var expirationDate time.Time
	var updatePassword types.UInt64

	err := manager.Database.QueryRow(`
		SELECT
			data_id,
			owner,
			size,
			name,
			data_type,
			meta_binary,
			access_permission,
			access_permission_recipients,
			update_permission,
			update_permission_recipients,
			creation_date,
			update_date,
			expiration_days,
			status,
			reference_count,
			refer_data_id,
			raw_flags,
			last_reference_date,
			expiration_date,
			tags,
			update_password
		FROM datastore.objects
		WHERE data_id=$1`, dataID).Scan(
		&metaInfo.DataID,
		&metaInfo.OwnerID,
		&metaInfo.Size,
		&metaInfo.Name,
		&metaInfo.DataType,
		&metaInfo.MetaBinary,
		&metaInfo.Permission.Permission,
		&metaInfo.Permission.RecipientIDs,
		&metaInfo.DelPermission.Permission,
		&metaInfo.DelPermission.RecipientIDs,
		&creationDate,
		&updateDate,
		&metaInfo.Period,
		&metaInfo.Status,
		&metaInfo.ReferredCnt,
		&metaInfo.ReferDataID,
		&metaInfo.Flag,
		&lastReferenceDate,
		&expirationDate,
		&metaInfo.Tags,
		&updatePassword,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return metaInfo, updatePassword, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return metaInfo, updatePassword, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	metaInfo.CreatedTime.FromTimestamp(creationDate)
	metaInfo.UpdatedTime.FromTimestamp(updateDate)
	metaInfo.ReferredTime.FromTimestamp(lastReferenceDate)
	metaInfo.ExpireTime.FromTimestamp(expirationDate)

	return metaInfo, updatePassword, nil
}
