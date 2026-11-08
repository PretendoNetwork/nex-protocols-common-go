package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func GetSearchResultsByDataIDsWithResultOption(database common_globals.Database, dataIDs types.List[types.UInt64], resultOption datastore_constants.ResultFlag) (types.List[datastore_types.DataStoreMetaInfo], *nex.Error) {
	// * This differs from a generic GetObjectMetaInfsoByDataIDsWithResultOption
	// * because we need to make sure the objects always come back in DataID
	// * order and they never have permission recipients populated even when
	// * ResultFlagPermittedIDs is set

	metaInfos := types.NewList[datastore_types.DataStoreMetaInfo]()

	populateTags := resultOption.HasFlag(datastore_constants.ResultFlagTags)
	populateRatings := resultOption.HasFlag(datastore_constants.ResultFlagRatings)
	populateMetaBinary := resultOption.HasFlag(datastore_constants.ResultFlagMetaBinary)

	rows, err := database.Query(`
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
			CASE WHEN $3 THEN meta_binary END
		FROM unnest($1::numeric[]) WITH ORDINALITY AS ids(data_id, position)
		JOIN datastore.objects o ON o.data_id = ids.data_id
		JOIN datastore.object_versions v ON v.data_id = o.data_id AND v.version = (SELECT MAX(version) FROM datastore.object_versions WHERE data_id = o.data_id AND completed = true)
		ORDER BY ids.position
	`, pq.Array(dataIDs), populateTags, populateMetaBinary)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		metaInfo := datastore_types.NewDataStoreMetaInfo()

		err := rows.Scan(
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
		)
		if err != nil {
			// TODO - Send more specific errors?
			return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		metaInfos = append(metaInfos, metaInfo)
	}

	if err := rows.Err(); err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	rows.Close()

	if populateRatings {
		for i := range metaInfos {
			ratings, errCode := GetObjectRatingsByDataID(database, metaInfos[i].DataID)
			if errCode != nil {
				return nil, errCode
			}

			metaInfos[i].Ratings = ratings
		}
	}

	return metaInfos, nil
}
