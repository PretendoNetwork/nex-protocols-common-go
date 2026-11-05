package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetObjectMetaInfoByDataIDWithResultOption(database common_globals.Database, dataID types.UInt64, resultOption datastore_constants.ResultFlag) (datastore_types.DataStoreMetaInfo, *nex.Error) {
	metaInfo := datastore_types.NewDataStoreMetaInfo()

	populateTags := resultOption.HasFlag(datastore_constants.ResultFlagTags)
	populateRatings := resultOption.HasFlag(datastore_constants.ResultFlagRatings)
	populateMetaBinary := resultOption.HasFlag(datastore_constants.ResultFlagMetaBinary)
	populateRecipientIDs := resultOption.HasFlag(datastore_constants.ResultFlagPermittedIDs)

	err := database.QueryRow(`
		SELECT
			o.data_id,
			owner,
			v.size,
			name,
			data_type,
			access_permission,
			update_permission,
			o.creation_date,
			update_date,
			expiration_days,
			status,
			reference_count,
			refer_data_id,
			raw_flags,
			last_reference_date,
			expiration_date,
			CASE WHEN $2 THEN tags END,
			CASE WHEN $3 THEN meta_binary END,
			CASE WHEN $4 THEN access_permission_recipients END,
			CASE WHEN $4 THEN update_permission_recipients END
		FROM datastore.objects o
		JOIN datastore.object_versions v ON v.data_id = o.data_id AND v.version = (SELECT MAX(version) FROM datastore.object_versions WHERE data_id = o.data_id AND completed = true)
		WHERE o.data_id = $1
	`, dataID, populateTags, populateMetaBinary, populateRecipientIDs).Scan(
		&metaInfo.DataID,
		&metaInfo.OwnerID,
		&metaInfo.Size,
		&metaInfo.Name,
		&metaInfo.DataType,
		&metaInfo.Permission.Permission,
		&metaInfo.DelPermission.Permission,
		&metaInfo.CreatedTime,
		&metaInfo.UpdatedTime,
		&metaInfo.Period,
		&metaInfo.Status,
		&metaInfo.ReferredCnt,
		&metaInfo.ReferDataID,
		&metaInfo.Flag,
		&metaInfo.ReferredTime,
		&metaInfo.ExpireTime,
		&metaInfo.Tags,
		&metaInfo.MetaBinary,
		&metaInfo.Permission.RecipientIDs,
		&metaInfo.DelPermission.RecipientIDs,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return metaInfo, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return metaInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if populateRatings {
		ratings, err := GetObjectRatingsByDataID(database, dataID)
		if err != nil {
			// TODO - Send more specific errors?
			return metaInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		metaInfo.Ratings = ratings
	}

	return metaInfo, nil
}
