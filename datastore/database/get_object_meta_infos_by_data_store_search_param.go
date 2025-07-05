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

// TODO:
// - OwnerIDs
// - OwnerType
// - DestinationIDs
// - CreatedAfter/CreatedBefore/UpdatedAfter/UpdatedBefore
// - ReferDataId
// - Tags
// - ResultOrder/ResultOrderColumn/ResultRange
// - MinimalRatingFrequency
// - UseCache
// - DataTypes

func GetObjectMetaInfosByDataStoreSearchParam(manager *common_globals.DataStoreManager, param datastore_types.DataStoreSearchParam, callerPID types.PID) ([]datastore_types.DataStoreMetaInfo, uint32, *nex.Error) {
	metaInfos := []datastore_types.DataStoreMetaInfo{}

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

	populateTags := (param.ResultOption & types.UInt8(datastore_constants.ResultFlagTags)) != 0
	populateRatings := (param.ResultOption & types.UInt8(datastore_constants.ResultFlagRatings)) != 0
	populateMetaBinary := (param.ResultOption & types.UInt8(datastore_constants.ResultFlagMetaBinary)) != 0
	populateRecipientIDs := (param.ResultOption & types.UInt8(datastore_constants.ResultFlagPermittedIDs)) != 0

	if populateTags {
		columns = append(columns, "tags")
	}

	if populateMetaBinary {
		columns = append(columns, "meta_binary")
	}

	if populateRecipientIDs {
		columns = append(columns, "access_permission_recipients")
		columns = append(columns, "update_permission_recipients")
	}

	friends := []uint32{}
	if manager.GetUserFriendPIDs != nil {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friends = manager.GetUserFriendPIDs(uint32(callerPID))
	} else {
		common_globals.Logger.Error("GetUserFriendPIDs not implemented! Assuming no friends.")
	}

	query := Select(columns...).From("datastore.objects")
	switch datastore_constants.SearchType(param.SearchTarget) {
	case datastore_constants.SearchTypePublic:
		query = query.
			Where("access_permission").Is(datastore_constants.PermissionPublic)
	case datastore_constants.SearchTypeSendFriend:
		query = query.
			Where("owner").Is(callerPID).
			And("access_permission").Is(datastore_constants.PermissionFriend)
	case datastore_constants.SearchTypeSendSpecified:
		query = query.
			Where("owner").Is(callerPID).
			And("access_permission").Is(datastore_constants.PermissionSpecified)
	case datastore_constants.SearchTypeSendSpecifiedFriend:
		query = query.
			Where("owner").Is(callerPID).
			And("access_permission").Is(datastore_constants.PermissionSpecifiedFriend)
	case datastore_constants.SearchTypeSend:
		query = query.
			Where("owner").Is(callerPID).
			And("access_permission").IsAnyOf([]datastore_constants.Permission{
			datastore_constants.PermissionFriend,
			datastore_constants.PermissionSpecified,
			datastore_constants.PermissionSpecifiedFriend,
		})
	case datastore_constants.SearchTypeFriend:
		query = query.
			Where("owner").IsAnyOf(friends).
			And("access_permission").Is(datastore_constants.PermissionFriend)
	case datastore_constants.SearchTypeReceivedSpecified:
		query = query.
			Where("access_permission").IsAnyOf([]datastore_constants.Permission{
			datastore_constants.PermissionSpecified,
			datastore_constants.PermissionSpecifiedFriend,
		}).
			And("access_permission_recipients").ArrayContains(callerPID)
	case datastore_constants.SearchTypeReceived:
		// TODO are you fucking serious
		common_globals.Logger.Warning("SearchTypeReceived unimplemented!")
	case datastore_constants.SearchTypePrivate:
		query = query.
			Where("owner").Is(callerPID).
			And("access_permission").Is(datastore_constants.PermissionPrivate)
	case datastore_constants.SearchTypeOwn:
		query = query.
			Where("owner").Is(callerPID).
			And("status").Is(datastore_constants.DataStatusNone)
	case datastore_constants.SearchTypePublicExcludeOwnAndFriends:
		// TODO aaaaaaa
		common_globals.Logger.Warning("SearchTypePublicExcludeOwnAndFriends unimplemented!")
	case datastore_constants.SearchTypeOwnPending:
		query = query.
			Where("owner").Is(callerPID).
			And("status").Is(datastore_constants.DataStatusPending)
	case datastore_constants.SearchTypeOwnRejected:
		query = query.
			Where("owner").Is(callerPID).
			And("status").Is(datastore_constants.DataStatusRejected)
	case datastore_constants.SearchTypeOwnAll:
		query = query.
			Where("owner").Is(callerPID)
	default:
		common_globals.Logger.Errorf("Unknown SearchTarget %v", param.SearchTarget)
	}

	switch datastore_constants.SearchTarget(param.OwnerType) {
	case datastore_constants.SearchTargetAnybody:
		if len(param.OwnerIDs) > 0 {
			query = query.Where("owner").IsAnyOf(param.OwnerIDs)
		}
	case datastore_constants.SearchTargetFriend:
		query = query.Where("owner").IsAnyOf(friends)
	case datastore_constants.SearchTargetAnybodyExcludeSpecified:
		// TODO aaaaaaa
		common_globals.Logger.Warning("SearchTargetAnybodyExcludeSpecified unimplemented!")
	default:
		common_globals.Logger.Errorf("Unknown OwnerType %v", param.OwnerType)
	}

	if param.DataType != 65535 {
		query = query.Where("data_type").Is(param.DataType)
	}

	query = query.Offset(int(param.ResultRange.Offset)).Limit(int(param.ResultRange.Length))

	final_query, params, err := query.Build()
	if err != nil {
		// TODO - Send more specific errors?
		return metaInfos, 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	common_globals.Logger.Info(final_query)
	common_globals.Logger.Infof("%v", params)
	rows, err := manager.Database.Query(final_query, params...)
	if err != nil {
		if err == sql.ErrNoRows {
			// Dev note says return [] here, not NotFound
			return metaInfos, 0, nil
		}

		// TODO - Send more specific errors?
		return metaInfos, 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		metaInfo := datastore_types.NewDataStoreMetaInfo()

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

		if populateTags {
			outs = append(outs, pq.Array(&metaInfo.Tags))
		}

		if populateMetaBinary {
			outs = append(outs, &metaInfo.MetaBinary)
		}

		if populateRecipientIDs {
			outs = append(outs, pq.Array(&metaInfo.Permission.RecipientIDs))
			outs = append(outs, pq.Array(&metaInfo.DelPermission.RecipientIDs))
		}

		err := rows.Scan(outs...)
		if err != nil {
			// TODO - Send more specific errors?
			return metaInfos, 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		if populateRatings {
			ratings, err := GetObjectRatingsByDataID(manager, metaInfo.DataID)
			if err != nil {
				// TODO - Send more specific errors?
				return metaInfos, 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
			}

			metaInfo.Ratings = ratings
		}

		metaInfos = append(metaInfos, metaInfo)
	}

	// TODO totalCount
	return metaInfos, uint32(len(metaInfos)), nil
}
