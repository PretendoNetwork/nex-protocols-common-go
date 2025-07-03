package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func GetObjectMetaInfoByDataIDWithResultOption(manager *common_globals.DataStoreManager, dataID types.UInt64, resultOption types.UInt8) (datastore_types.DataStoreMetaInfo, *nex.Error) {
	metaInfo := datastore_types.NewDataStoreMetaInfo()

	// * These fields are always populated.
	// * Tags, ratings, meta binary, and
	// * permission recipient IDs are only
	// * populated if resultOption says to
	columns := []string{
		"data_id",
		"owner",
		"size",
		"name",
		"data_type",
		"access_permission",
		"update_permission",
		"creation_date",
		"update_date",
		"expiration_days",
		"status",
		"reference_count",
		"refer_data_id",
		"raw_flags",
		"last_reference_date",
		"expiration_date",
	}

	outs := []any{
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
	}

	populateTags := (resultOption & types.UInt8(datastore_constants.ResultFlagTags)) != 0
	populateRatings := (resultOption & types.UInt8(datastore_constants.ResultFlagRatings)) != 0
	populateMetaBinary := (resultOption & types.UInt8(datastore_constants.ResultFlagMetaBinary)) != 0
	populateRecipientIDs := (resultOption & types.UInt8(datastore_constants.ResultFlagPermittedIDs)) != 0

	if populateTags {
		columns = append(columns, "tags")
		outs = append(outs, pq.Array(&metaInfo.Tags))
	}

	if populateMetaBinary {
		columns = append(columns, "meta_binary")
		outs = append(outs, pq.Array(&metaInfo.MetaBinary))
	}

	if populateRecipientIDs {
		columns = append(columns, "access_permission_recipients")
		columns = append(columns, "update_permission_recipients")
		outs = append(outs, pq.Array(&metaInfo.Permission.RecipientIDs))
		outs = append(outs, pq.Array(&metaInfo.DelPermission.RecipientIDs))
	}

	query, params, err := Select(columns...).From("datastore.objects").Where("data_id").Is(dataID).Build()
	if err != nil {
		// TODO - Send more specific errors?
		return metaInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	err = manager.Database.QueryRow(query, params...).Scan(outs...)
	if err != nil {
		if err == sql.ErrNoRows {
			return metaInfo, nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return metaInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if populateRatings {
		ratings, err := GetObjectRatingsByDataID(manager, dataID)
		if err != nil {
			// TODO - Send more specific errors?
			return metaInfo, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		metaInfo.Ratings = ratings
	}

	return metaInfo, nil
}
