package database

import (
	"slices"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func UpdateObjectMetadata(database common_globals.Database, currentData datastore_types.DataStoreMetaInfo, newData datastore_types.DataStoreChangeMetaParam) *nex.Error {
	// * Do nothing if nothing to update, besides the comparison
	if newData.ModifiesFlag == datastore_constants.ModificationFlagNone {
		return compareObjectMetas(currentData, newData.CompareParam)
	}

	modifyName := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagName)
	modifyAccessPermission := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagAccessPermission)
	modifyUpdatePermission := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagUpdatePermission)
	modifyPeriod := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagPeriod)
	modifyMetaBinary := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagMetaBinary)
	modifyTags := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagTags)
	modifyUpdatedTime := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagUpdatedTime)
	modifyDataType := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagDataType)
	modifyReferredCount := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagReferredCount)
	modifyStatus := newData.ModifiesFlag.HasFlag(datastore_constants.ModificationFlagStatus)

	now := time.Now().UTC()
	updateData := map[string]any{
		"true_update_date": now, // * Always update this, since NEX only wants to update the "update time" in SOME instances
	}

	if modifyName {
		if utf8.RuneCountInString(string(newData.Name)) > datastore_constants.MaxNameLength {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a name which is too long")
		}

		updateData["name"] = newData.Name
	}

	if modifyAccessPermission {
		if newData.Permission.Permission > datastore_constants.PermissionSpecifiedFriend {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid access permission")
		}

		if len(newData.Permission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many access recipient IDs")
		}

		if newData.Permission.Permission != datastore_constants.PermissionSpecified && newData.Permission.Permission != datastore_constants.PermissionSpecifiedFriend && len(newData.Permission.RecipientIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with access recipient IDs on a permission which does not use them")
		}

		updateData["access_permission"] = newData.Permission.Permission
		updateData["access_permission_recipients"] = pq.Array(newData.Permission.RecipientIDs)
	}

	if modifyUpdatePermission {
		if newData.DelPermission.Permission > datastore_constants.PermissionSpecifiedFriend {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid update permission")
		}

		if len(newData.DelPermission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many update recipient IDs")
		}

		if newData.DelPermission.Permission != datastore_constants.PermissionSpecified && newData.DelPermission.Permission != datastore_constants.PermissionSpecifiedFriend && len(newData.DelPermission.RecipientIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with update recipient IDs on a permission which does not use them")
		}

		updateData["update_permission"] = newData.DelPermission.Permission
		updateData["update_permission_recipients"] = pq.Array(newData.DelPermission.RecipientIDs)
	}

	if modifyPeriod {
		// * Does not seem to be a constant in DataStore, so just inline it for now?
		// * A period of 0 is valid and just immediately expires the object
		if newData.Period > 365 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid expiration days")
		}

		updateData["expiration_days"] = newData.Period
	}

	if modifyMetaBinary {
		if len(newData.MetaBinary) > datastore_constants.MaxMetaBinSize {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a MetaBinary which is too long")
		}

		// * Objects created with an empty MetaBinary have the field set to NULL for GetSpecificMeta
		// * but when the object is updated, even if the MetaBinary is also empty, it gets treated
		// * differently. So undo the NULL here even if the value is still "empty"
		updateData["meta_binary"] = append([]byte{}, newData.MetaBinary...)
	}

	if modifyTags {
		if len(newData.Tags) > int(datastore_constants.NumTagSlot) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many tags")
		}

		seenTags := make([]string, 0)
		for _, tag := range newData.Tags {
			if utf8.RuneCountInString(string(tag)) > datastore_constants.MaxTagLength {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a tag which is too long")
			}

			if strings.ContainsFunc(string(tag), unicode.IsControl) {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a tag containing control characters")
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

	if modifyReferredCount {
		updateData["reference_count"] = newData.ReferredCnt
	}

	if modifyStatus {
		// * Statuses 1, 3 and 4 seem to exist but have no known names.
		// * Servers accept them, but no client (not even the owner) can
		// * ever interact with the objects again when set
		if newData.Status > datastore_constants.DataStatusRejected {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
		}

		// * Unofficial behaviour, but I see no reason to implement
		// * unless we know a client needs it. Allowing the status
		// * to be updated would mean owners could bypass review checks
		// * by setting the status to DataStatusNone
		if currentData.Status != datastore_constants.DataStatusNone {
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
		}

		updateData["status"] = newData.Status
	}

	// * Only compare after the new values are check for validity
	if errCode := compareObjectMetas(currentData, newData.CompareParam); errCode != nil {
		return errCode
	}

	// * This happens regardless of if DATA_FLAG_PERIOD_FROM_LAST_REFERRED
	// * is set or not. DATA_FLAG_PERIOD_FROM_LAST_REFERRED seems to only
	// * apply to PrepareGetObject (and related) and TouchObject
	day := 24 * time.Hour

	// * Persisted objects don't get their expiration updated
	if !currentData.ExpireTime.IsPermanentDateTime() {
		if modifyUpdatedTime && modifyPeriod {
			updateData["expiration_date"] = now.Add(time.Duration(newData.Period) * day)
		} else if modifyUpdatedTime {
			updateData["expiration_date"] = now.Add(time.Duration(currentData.Period) * day)
		} else if modifyPeriod {
			updateData["expiration_date"] = currentData.UpdatedTime.Standard().UTC().Add(time.Duration(newData.Period) * day)
		}
	}

	query, params, err := Update("datastore.objects").Set(updateData).Where("data_id").Is(currentData.DataID).Build()
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	_, err = database.Exec(query, params...)
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}

func compareObjectMetas(currentData datastore_types.DataStoreMetaInfo, compareParam datastore_types.DataStoreChangeMetaCompareParam) *nex.Error {
	compareName := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagName)
	compareAccessPermission := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagAccessPermission)
	compareUpdatePermission := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagUpdatePermission)
	comparePeriod := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagPeriod)
	compareMetaBinary := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagMetaBinary)
	compareTags := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagTags)
	compareDataType := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagDataType)
	compareReferredCount := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagReferredCount)
	compareStatus := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagStatus)

	if compareParam.ComparisonFlag == datastore_constants.ComparisonFlagAll {
		compareName = true
		compareAccessPermission = true
		compareUpdatePermission = true
		comparePeriod = true
		compareMetaBinary = true
		compareTags = true
		compareDataType = true
		compareReferredCount = true
		compareStatus = true
	}

	if compareName && !currentData.Name.Equals(compareParam.Name) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareAccessPermission && !currentData.Permission.Equals(compareParam.Permission) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareUpdatePermission && !currentData.DelPermission.Equals(compareParam.DelPermission) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if comparePeriod && !currentData.Period.Equals(compareParam.Period) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareMetaBinary && !currentData.MetaBinary.Equals(compareParam.MetaBinary) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareTags {
		currentTags := slices.Sorted(slices.Values(currentData.Tags))
		compareTags := slices.Sorted(slices.Values(compareParam.Tags))

		if !slices.Equal(currentTags, compareTags) {
			return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
		}
	}

	if compareDataType && !currentData.DataType.Equals(compareParam.DataType) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareReferredCount && !currentData.ReferredCnt.Equals(compareParam.ReferredCnt) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareStatus && currentData.Status != compareParam.Status {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	return nil
}
