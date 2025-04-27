package database

import (
	"math/rand/v2"
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

func InsertObjectByPreparePostParam(manager *common_globals.DataStoreManager, ownerPID types.PID, param datastore_types.DataStorePreparePostParam) (uint64, *nex.Error) {
	// * Max size is 10MiB.
	// * Does not seem to be a constant in DataStore, so just inline it for now?
	if param.Size > 0xA00000 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object larger than 10MiB")
	}

	if len(param.Name) > int(datastore_constants.MaxNameLength) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a name which is too long")
	}

	if param.DataType == types.UInt16(datastore_constants.InvalidDataType) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid DataType")
	}

	if len(param.MetaBinary) > int(datastore_constants.MaxMetaBinSize) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a MetaBinary which is too long")
	}

	if param.Permission.Permission > types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid access permission")
	}

	if len(param.Permission.RecipientIDs) > int(datastore_constants.DatastorePermissionRecipientIDsMax) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many access recipient IDs")
	}

	if param.DelPermission.Permission > types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid update permission")
	}

	if len(param.DelPermission.RecipientIDs) > int(datastore_constants.DatastorePermissionRecipientIDsMax) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many update recipient IDs")
	}

	if len(param.Tags) > int(datastore_constants.NumTagSlot) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many tags")
	}

	seenTags := make([]string, 0)
	for _, tag := range param.Tags {
		if len(tag) > int(datastore_constants.MaxTagLength) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a tag which is too long")
		}

		if slices.Contains(seenTags, string(tag)) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with duplicate tags")
		}

		seenTags = append(seenTags, string(tag))
	}

	// * Does not seem to be a constant in DataStore, so just inline it for now?
	if param.Period == 0 || param.Period > 365 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid expiration days")
	}

	if len(param.RatingInitParams) > int(datastore_constants.NumRatingSlot) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many RatingInitParams")
	}

	seenRatingSlots := make([]int8, 0)
	for _, ratingInitParam := range param.RatingInitParams {
		// * Sent as an SInt8, but only values 0-15 are allowed? Why Nintendo?
		if ratingInitParam.Slot < 0 || ratingInitParam.Slot > types.Int8(datastore_constants.RatingSlotMax) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating slot")
		}

		if ratingInitParam.Param.LockType > types.UInt8(datastore_constants.RatingLockPermanent) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating lock type")
		}

		// * Different lock types interpret the lock values differently

		// * "Interval" locks treat PeriodDuration as a non-negative value representing
		// * the number of seconds until the lock expires
		if ratingInitParam.Param.LockType == types.UInt8(datastore_constants.RatingLockInterval) {
			if ratingInitParam.Param.PeriodDuration < 0 {
				return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}
		}

		// * "Period" locks treat PeriodDuration as the day of the week/month and
		// * PeriodHour as the hour of that day. "Day1" is the first of the following
		// * month
		if ratingInitParam.Param.LockType == types.UInt8(datastore_constants.RatingLockPeriod) {
			if ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodMon) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodTue) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodWed) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodThu) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodFri) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodSat) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodSun) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodDay1) {
				return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}

			// * Sent as an SInt8, I wonder if "negative" time is possible?
			// * Like for referencing days in the past? This would allow the
			// * client to target the LAST day of the month as well as the first?
			if ratingInitParam.Param.PeriodHour < 0 || ratingInitParam.Param.PeriodHour > 23 {
				return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodHour")
			}
		}

		if slices.Contains(seenRatingSlots, int8(ratingInitParam.Slot)) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with duplicate rating slot")
		}

		seenRatingSlots = append(seenRatingSlots, int8(ratingInitParam.Slot))
	}

	// * Slot IDs can only be between 0-15, and 0xFFFF. Despite it calling 0xFFFF
	// * "invalid", this actually indicates that the object does not use persistence
	if param.PersistenceInitParam.PersistenceSlotID > types.UInt16(datastore_constants.NumPersistenceSlot-1) &&
		param.PersistenceInitParam.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many RatingInitParams")
	}

	if err := manager.ValidateExtraData(param.ExtraData); err != nil {
		return 0, nil
	}

	var dataID uint64

	sortedTags := make([]string, 0, len(param.Tags))
	for i := range param.Tags {
		sortedTags = append(sortedTags, string(param.Tags[i]))
	}

	// * Tags get re-ordered in alphabetical order
	sort.Strings(sortedTags)

	extraData := make([]string, 0, len(param.ExtraData))
	for i := range param.Tags {
		extraData = append(extraData, string(param.ExtraData[i]))
	}

	needsReview := (param.Flag & types.UInt32(datastore_constants.DataFlagNeedReview)) != 0
	updateExpirationOnReference := (param.Flag & types.UInt32(datastore_constants.DataFlagPeriodFromLastReferred)) != 0
	useReadLock := (param.Flag & types.UInt32(datastore_constants.DataFlagUseReadLock)) != 0
	notifyAccessRecipientsOnCreation := (param.Flag & types.UInt32(datastore_constants.DataFlagUseNotificationOnPost)) != 0
	notifyAccessRecipientsOnUpdate := (param.Flag & types.UInt32(datastore_constants.DataFlagUseNotificationOnUpdate)) != 0
	notUseFileServer := (param.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	needUploadCompletion := (param.Flag & types.UInt32(datastore_constants.DataFlagNeedCompletion)) != 0

	status := datastore_constants.DataStatusNone
	if needsReview {
		status = datastore_constants.DataStatusPending
	}

	uploadCompleted := false
	if !needUploadCompletion {
		uploadCompleted = true
	}

	// * These are always generated by the server.
	// * Only the object owner can request them.
	// * Using an objects password will bypass
	// * certain permission checks, do not give
	// * these out freely
	accessPassword := rand.Uint64()
	updatePassword := rand.Uint64()

	// * Persistent objects are set to never expire
	expirationDate := time.Now().UTC().Add(time.Duration(param.Period) * 24 * time.Hour)
	if param.PersistenceInitParam.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		expirationDate = time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC)
	}

	err := manager.Database.QueryRow(`INSERT INTO datastore.objects (
		owner,
		size,
		name,
		data_type,
		meta_binary,
		access_permission,
		access_permission_recipients,
		update_permission,
		update_permission_recipients,
		raw_flags,
		expiration_days,
		refer_data_id,
		tags,
		persistence_slot_id,
		extra_data,
		needs_review,
		update_expiration_on_reference,
		use_read_lock,
		notify_access_recipients_on_creation,
		notify_access_recipients_on_update,
		not_use_file_server,
		need_upload_completion,
		status,
		access_password,
		update_password,
		expiration_date,
		upload_completed
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		$10,
		$11,
		$12,
		$13,
		$14,
		$15,
		$16,
		$17,
		$18,
		$19,
		$20,
		$21,
		$22,
		$23,
		$24,
		$25,
		$26,
		$27
	) RETURNING data_id`,
		ownerPID,
		param.Size,
		param.Name,
		param.DataType,
		param.MetaBinary,
		param.Permission.Permission,
		pq.Array(param.Permission.RecipientIDs),
		param.DelPermission.Permission,
		pq.Array(param.DelPermission.RecipientIDs),
		param.Flag,
		param.Period,
		param.ReferDataID,
		pq.Array(sortedTags),
		param.PersistenceInitParam.PersistenceSlotID,
		pq.Array(extraData),
		needsReview,
		updateExpirationOnReference,
		useReadLock,
		notifyAccessRecipientsOnCreation,
		notifyAccessRecipientsOnUpdate,
		notUseFileServer,
		needUploadCompletion,
		status,
		accessPassword,
		updatePassword,
		expirationDate,
		uploadCompleted,
	).Scan(&dataID)

	if err != nil {
		// TODO - Send more specific errors?
		return 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return dataID, nil
}
