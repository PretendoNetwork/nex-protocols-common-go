package database

import (
	"math/rand/v2"
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

func InsertObjectByPreparePostParam(database common_globals.Database, manager *common_globals.DataStoreManager, ownerPID types.PID, param datastore_types.DataStorePreparePostParam) (uint64, *nex.Error) {
	// * Max size is 10MiB.
	// * Does not seem to be a constant in DataStore, so just inline it for now?
	if param.Size > 0xA00000 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object larger than 10MiB")
	}

	if utf8.RuneCountInString(string(param.Name)) > datastore_constants.MaxNameLength {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a name which is too long")
	}

	if param.DataType == types.UInt16(datastore_constants.InvalidDataType) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid DataType")
	}

	if len(param.MetaBinary) > datastore_constants.MaxMetaBinSize {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a MetaBinary which is too long")
	}

	if param.Permission.Permission > datastore_constants.PermissionSpecifiedFriend {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid access permission")
	}

	// TODO - Pokemon Bank does not validate this, was this added in a later NEX version? Keeping this around just in case, makes no sense to have this constant and not use it?
	/*
		if len(param.Permission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many access recipient IDs")
		}
	*/

	if param.Permission.Permission != datastore_constants.PermissionSpecified && param.Permission.Permission != datastore_constants.PermissionSpecifiedFriend && len(param.Permission.RecipientIDs) != 0 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with access recipient IDs on a permission which does not use them")
	}

	if param.DelPermission.Permission > datastore_constants.PermissionSpecifiedFriend {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid update permission")
	}

	// TODO - Pokemon Bank does not validate this, was this added in a later NEX version? Keeping this around just in case, makes no sense to have this constant and not use it?
	/*
		if len(param.DelPermission.RecipientIDs) > datastore_constants.DatastorePermissionRecipientIDsMax {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many update recipient IDs")
		}
	*/

	if param.DelPermission.Permission != datastore_constants.PermissionSpecified && param.DelPermission.Permission != datastore_constants.PermissionSpecifiedFriend && len(param.DelPermission.RecipientIDs) != 0 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with update recipient IDs on a permission which does not use them")
	}

	if len(param.Tags) > int(datastore_constants.NumTagSlot) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many tags")
	}

	seenTags := make([]string, 0)
	for _, tag := range param.Tags {
		if utf8.RuneCountInString(string(tag)) > datastore_constants.MaxTagLength {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a tag which is too long")
		}

		if strings.ContainsFunc(string(tag), unicode.IsControl) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with a tag containing control characters")
		}

		if slices.Contains(seenTags, string(tag)) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with duplicate tags")
		}

		seenTags = append(seenTags, string(tag))
	}

	// * Does not seem to be a constant in DataStore, so just inline it for now?
	// * A period of 0 is valid and just immediately expires the object
	if param.Period > 365 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with invalid expiration days")
	}

	if len(param.RatingInitParams) > int(datastore_constants.NumRatingSlot) {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many RatingInitParams")
	}

	seenRatingSlots := make([]int8, 0)
	for _, ratingInitParam := range param.RatingInitParams {
		errCode := validateRatingInitParam(ratingInitParam)
		if errCode != nil {
			return 0, errCode
		}

		if slices.Contains(seenRatingSlots, int8(ratingInitParam.Slot)) {
			return 0, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "Tried to upload object with duplicate rating slot")
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
		return 0, err
	}

	knownFlags := datastore_constants.DataFlagNeedReview | datastore_constants.DataFlagPeriodFromLastReferred | datastore_constants.DataFlagUseReadLock | datastore_constants.DataFlagUseNotificationOnPost | datastore_constants.DataFlagUseNotificationOnUpdate | datastore_constants.DataFlagNotUseFileServer | datastore_constants.DataFlagNeedCompletion

	// TODO - It IS possible this is wrong. This was checked on Pokemon Bank, which customizes DataStore, so "unknown flags" COULD be valid flags we don't understand in Bank. Need to check on a stock server
	if !manager.AllowUnknownObjectFlags && param.Flag&^knownFlags != 0 {
		return 0, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with unknown flags")
	}

	needsReview := param.Flag.HasFlag(datastore_constants.DataFlagNeedReview)
	updateExpirationOnReference := param.Flag.HasFlag(datastore_constants.DataFlagPeriodFromLastReferred)
	useReadLock := param.Flag.HasFlag(datastore_constants.DataFlagUseReadLock)
	notifyAccessRecipientsOnCreation := param.Flag.HasFlag(datastore_constants.DataFlagUseNotificationOnPost)
	notifyAccessRecipientsOnUpdate := param.Flag.HasFlag(datastore_constants.DataFlagUseNotificationOnUpdate)
	notUseFileServer := param.Flag.HasFlag(datastore_constants.DataFlagNotUseFileServer)
	needUploadCompletion := param.Flag.HasFlag(datastore_constants.DataFlagNeedCompletion)

	// * I believe the useReadLock flag is related to the "lockId" seen
	// * on DataStorePrepareGetParamV1, DataStorePrepareGetParam and
	// * DataStoreTouchObjectParam. My hunch is that this is related
	// * to locking down the download of an object from the file server
	// * based on the field/flag names and how similar they are to each
	// * other, as well as the contexts in when they are used
	// *
	// * My thinking:
	// *
	// * 1. "lockId" is ONLY ever referenced in methods related to
	// *    downloading an object from the file server
	// * 2. The "lockId" is never exposed by the server, so I believe
	// *    it is set by the client and then saved by the server later
	// * 3. In other protocols, like Ranking, sometimes settings are
	// *    defined on the first call of a method. For example in
	// *    the Ranking protocol the order (ascending/descending) of
	// *    which rankings are returned in GetRanking is not configured
	// *    until the first call to UploadScore is made. When the first
	// *    call to UploadScore is made, the server sets the order of
	// *    rankings to use what was in RankingScoreData.orderBy
	// * 4. Since other protocols do not set certain settings until
	// *    a client calls the method for the first time, and since
	// *    the "lockId" is only seen in client requests, I believe
	// *    that the objects "lockId" is configured the first time
	// *    the client sends either a DataStorePrepareGetParamV1,
	// *    DataStorePrepareGetParam or DataStoreTouchObjectParam
	// * 5. Logically then, assuming the above is correct, it would
	// *    make sense that the "lockId" would act like a password,
	// *    preventing users who do not have the "lockId" from
	// *    downloading the object data from the file server. This
	// *    could have real legitimate uses, such as displaying an
	// *    objects metadata (by having access permissions set to public)
	// *    but blocking the download of the objects data from the
	// *    file server until a condition is met. This could be used
	// *    for things like player-owned item shops, letting players
	// *    display goods to the public but only allowing downloads
	// *    once another player has bought an item and the seller has
	// *    communicated the "lockId" (either manually or automatically
	// *    via the games implementation)

	status := datastore_constants.DataStatusNone
	if needsReview {
		status = datastore_constants.DataStatusPending
	}

	// * Objects which use the file server are never available until the
	// * upload has been completed. However objects which do not use the
	// * file server can be made available immediately, unless the flag
	// * DATA_FLAG_NEED_COMPLETION is set
	uploadCompleted := notUseFileServer && !needUploadCompletion

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
		expirationDate = types.NewPermanentDateTime().Standard()
	}

	var dataID uint64

	// * The object and its first version are inserted together. An empty
	// * MetaBinary is stored as NULL so that GetSpecificMeta can tell it
	// * apart from a MetaBinary that was later updated to have a size of 0
	// TODO - Ugly ass hacks man, we need to change the DB schema
	err := database.QueryRow(`WITH object AS (
		INSERT INTO datastore.objects (
			owner,
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
			expiration_date
		) VALUES (
			$1,
			$2,
			$3,
			NULLIF($4, ''::bytea),
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			ARRAY(SELECT tag FROM unnest($12::text[]) AS tag ORDER BY tag COLLATE "C"),
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
			$24
		) RETURNING data_id
	)
	INSERT INTO datastore.object_versions (
		data_id,
		version,
		size,
		extra_data,
		preparer,
		completed
	)
	SELECT data_id, 1, $25, $26, $1, $27 FROM object
	RETURNING data_id`,
		ownerPID,
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
		pq.Array(param.Tags),
		param.PersistenceInitParam.PersistenceSlotID,
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
		param.Size,
		pq.Array(param.ExtraData),
		uploadCompleted,
	).Scan(&dataID)

	if err != nil {
		// TODO - Send more specific errors?
		return 0, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return dataID, nil
}
