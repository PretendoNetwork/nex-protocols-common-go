package database

import (
	"slices"
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

func UpdateObjectMetadata(database common_globals.Database, manager *common_globals.DataStoreManager, currentData datastore_types.DataStoreMetaInfo, newData datastore_types.DataStoreChangeMetaParam) *nex.Error {
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

	if modifyName {
		if utf8.RuneCountInString(string(newData.Name)) > datastore_constants.MaxNameLength {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a name which is too long")
		}
	}

	if modifyAccessPermission {
		if newData.Permission.Permission > datastore_constants.PermissionSpecifiedFriend {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid access permission")
		}

		// TODO - Pokemon Bank does not validate this, was this added in a later NEX version? Keeping this around just in case, makes no sense to have this constant and not use it?
		/*
			if len(newData.Permission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many access recipient IDs")
			}
		*/

		if newData.Permission.Permission != datastore_constants.PermissionSpecified && newData.Permission.Permission != datastore_constants.PermissionSpecifiedFriend && len(newData.Permission.RecipientIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with access recipient IDs on a permission which does not use them")
		}
	}

	if modifyUpdatePermission {
		if newData.DelPermission.Permission > datastore_constants.PermissionSpecifiedFriend {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid update permission")
		}

		// TODO - Pokemon Bank does not validate this, was this added in a later NEX version? Keeping this around just in case, makes no sense to have this constant and not use it?
		/*
			if len(newData.DelPermission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with too many update recipient IDs")
			}
		*/

		if newData.DelPermission.Permission != datastore_constants.PermissionSpecified && newData.DelPermission.Permission != datastore_constants.PermissionSpecifiedFriend && len(newData.DelPermission.RecipientIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with update recipient IDs on a permission which does not use them")
		}
	}

	if modifyPeriod {
		// * Does not seem to be a constant in DataStore, so just inline it for now?
		// * A period of 0 is valid and just immediately expires the object
		if newData.Period > 365 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid expiration days")
		}
	}

	if modifyMetaBinary {
		if len(newData.MetaBinary) > datastore_constants.MaxMetaBinSize {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with a MetaBinary which is too long")
		}
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
	}

	if modifyDataType {
		if newData.DataType == types.UInt16(datastore_constants.InvalidDataType) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to update object with invalid DataType")
		}
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
		if !manager.AllowUnsafeObjectStatusUpdates && currentData.Status != datastore_constants.DataStatusNone {
			return nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
		}
	}

	// * Only compare after the new values are check for validity
	if errCode := compareObjectMetas(currentData, newData.CompareParam); errCode != nil {
		return errCode
	}

	// * This happens regardless of if DATA_FLAG_PERIOD_FROM_LAST_REFERRED
	// * is set or not. DATA_FLAG_PERIOD_FROM_LAST_REFERRED seems to only
	// * apply to PrepareGetObject (and related) and TouchObject
	day := 24 * time.Hour
	now := time.Now().UTC()

	// * Persisted objects don't get their expiration updated
	modifyExpirationDate := false
	var expirationDate time.Time

	if !currentData.ExpireTime.IsPermanentDateTime() {
		if modifyUpdatedTime && modifyPeriod {
			modifyExpirationDate = true
			expirationDate = now.Add(time.Duration(newData.Period) * day)
		} else if modifyUpdatedTime {
			modifyExpirationDate = true
			expirationDate = now.Add(time.Duration(currentData.Period) * day)
		} else if modifyPeriod {
			modifyExpirationDate = true
			expirationDate = currentData.UpdatedTime.Standard().UTC().Add(time.Duration(newData.Period) * day)
		}
	}

	_, err := database.Exec(`
		UPDATE datastore.objects SET
			true_update_date = $2,
			name = CASE WHEN $3 THEN $4 ELSE name END,
			access_permission = CASE WHEN $5 THEN $6 ELSE access_permission END,
			access_permission_recipients = CASE WHEN $5 THEN $7 ELSE access_permission_recipients END,
			update_permission = CASE WHEN $8 THEN $9 ELSE update_permission END,
			update_permission_recipients = CASE WHEN $8 THEN $10 ELSE update_permission_recipients END,
			expiration_days = CASE WHEN $11 THEN $12 ELSE expiration_days END,
			meta_binary = CASE WHEN $13 THEN $14 ELSE meta_binary END,
			tags = CASE WHEN $15 THEN ARRAY(SELECT tag FROM unnest($16::text[]) AS tag ORDER BY tag COLLATE "C") ELSE tags END,
			update_date = CASE WHEN $17 THEN $2 ELSE update_date END,
			data_type = CASE WHEN $18 THEN $19 ELSE data_type END,
			reference_count = CASE WHEN $20 THEN $21 ELSE reference_count END,
			status = CASE WHEN $22 THEN $23 ELSE status END,
			expiration_date = CASE WHEN $24 THEN $25 ELSE expiration_date END
		WHERE data_id = $1
	`,
		currentData.DataID,
		now,
		modifyName,
		newData.Name,
		modifyAccessPermission,
		newData.Permission.Permission,
		pq.Array(newData.Permission.RecipientIDs),
		modifyUpdatePermission,
		newData.DelPermission.Permission,
		pq.Array(newData.DelPermission.RecipientIDs),
		modifyPeriod,
		newData.Period,
		modifyMetaBinary,
		newData.MetaBinary,
		modifyTags,
		pq.Array(newData.Tags),
		modifyUpdatedTime,
		modifyDataType,
		newData.DataType,
		modifyReferredCount,
		newData.ReferredCnt,
		modifyStatus,
		newData.Status,
		modifyExpirationDate,
		expirationDate,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
