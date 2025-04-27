package database

import (
	"slices"
	"sort"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func UpdateObjectMetadata(manager *common_globals.DataStoreManager, currentData datastore_types.DataStoreMetaInfo, newData datastore_types.DataStoreChangeMetaParam) *nex.Error {
	// * Do nothing if nothing to update
	if newData.ModifiesFlag == types.UInt32(datastore_constants.ModificationFlagNone) {
		return nil
	}

	modifyName := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagName)) != 0
	modifyAccessPermission := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagAccessPermission)) != 0
	modifyUpdatePermission := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagUpdatePermission)) != 0
	modifyPeriod := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagPeriod)) != 0
	modifyMetaBinary := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagMetaBinary)) != 0
	modifyTags := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagTags)) != 0
	modifyUpdatedTime := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagUpdatedTime)) != 0
	modifyDataType := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagDataType)) != 0
	modifyStatus := (newData.ModifiesFlag & types.UInt32(datastore_constants.ModificationFlagStatus)) != 0

	now := time.Now().UTC()
	updateData := map[string]any{
		"true_update_date": now, // * Always update this, since NEX only wants to update the "update time" in SOME instances
	}

	if modifyName {
		if len(newData.Name) > int(datastore_constants.MaxNameLength) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a name which is too long")
		}

		updateData["name"] = newData.Name
	}

	if modifyAccessPermission {
		if newData.Permission.Permission > types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid access permission")
		}

		if len(newData.Permission.RecipientIDs) > int(datastore_constants.DatastorePermissionRecipientIDsMax) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many access recipient IDs")
		}

		updateData["access_permission"] = newData.Permission.Permission
		updateData["access_permission_recipients"] = pq.Array(newData.Permission.RecipientIDs)
	}

	if modifyUpdatePermission {
		if newData.DelPermission.Permission > types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid update permission")
		}

		if len(newData.DelPermission.RecipientIDs) > int(datastore_constants.DatastorePermissionRecipientIDsMax) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many update recipient IDs")
		}

		updateData["update_permission"] = newData.DelPermission.Permission
		updateData["update_permission_recipients"] = pq.Array(newData.DelPermission.RecipientIDs)
	}

	if modifyPeriod {
		// * Does not seem to be a constant in DataStore, so just inline it for now?
		if newData.Period == 0 || newData.Period > 365 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid expiration days")
		}

		updateData["expiration_days"] = newData.Period
	}

	if modifyMetaBinary {
		if len(newData.MetaBinary) > int(datastore_constants.MaxMetaBinSize) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a MetaBinary which is too long")
		}

		updateData["meta_binary"] = newData.MetaBinary
	}

	if modifyTags {
		if len(newData.Tags) > int(datastore_constants.NumTagSlot) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many tags")
		}

		seenTags := make([]string, 0)
		for _, tag := range newData.Tags {
			if len(tag) > int(datastore_constants.MaxTagLength) {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a tag which is too long")
			}

			if slices.Contains(seenTags, string(tag)) {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with duplicate tags")
			}

			seenTags = append(seenTags, string(tag))
		}

		sortedTags := make([]string, 0, len(newData.Tags))
		for i := range newData.Tags {
			sortedTags = append(sortedTags, string(newData.Tags[i]))
		}

		// * Tags get re-ordered in alphabetical order
		sort.Strings(sortedTags)

		updateData["tags"] = pq.Array(sortedTags)
	}

	if modifyUpdatedTime {
		updateData["update_date"] = now
	}

	if modifyDataType {
		if newData.DataType == types.UInt16(datastore_constants.InvalidDataType) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid DataType")
		}

		updateData["data_type"] = newData.DataType
	}

	if modifyStatus {
		if newData.Status != types.UInt8(datastore_constants.DataStatusNone) &&
			newData.Status != types.UInt8(datastore_constants.DataStatusPending) &&
			newData.Status != types.UInt8(datastore_constants.DataStatusRejected) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
		}

		// * Unofficial behaviour, but I see no reason to implement
		// * unless we know a client needs it. Allowing the status
		// * to be updated would mean owners could bypass review checks
		// * by setting the status to DataStatusNone
		if currentData.Status != types.UInt8(datastore_constants.DataStatusNone) {
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
		}

		updateData["status"] = newData.Status
	}

	// * This happens regardless of if DATA_FLAG_PERIOD_FROM_LAST_REFERRED
	// * is set or not. DATA_FLAG_PERIOD_FROM_LAST_REFERRED seems to only
	// * apply to PrepareGetObject (and related) and TouchObject
	day := 24 * time.Hour
	if modifyUpdatedTime && modifyPeriod {
		updateData["expiration_date"] = now.Add(time.Duration(newData.Period) * day)
	} else if modifyUpdatedTime {
		updateData["expiration_date"] = now.Add(time.Duration(currentData.Period) * day)
	} else if modifyPeriod {
		updateData["expiration_date"] = currentData.UpdatedTime.Standard().UTC().Add(time.Duration(newData.Period) * day)
	}

	query, params, err := Update("datastore.objects").Set(updateData).Where("data_id").Is(currentData.DataID).Build()
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	_, err = manager.Database.Exec(query, params...)
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
