package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	_ "github.com/PretendoNetwork/nex-protocols-go/v2"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

type CommonProtocol struct {
	endpoint                     nex.EndpointInterface
	protocol                     datastore.Interface
	manager                      *common_globals.DataStoreManager
	OnAfterDeleteObject          func(packet nex.PacketInterface, param datastore_types.DataStoreDeleteParam)
	OnAfterGetMeta               func(packet nex.PacketInterface, param datastore_types.DataStoreGetMetaParam)
	OnAfterGetMetas              func(packet nex.PacketInterface, dataIDs types.List[types.UInt64], param datastore_types.DataStoreGetMetaParam)
	OnAfterSearchObject          func(packet nex.PacketInterface, param datastore_types.DataStoreSearchParam)
	OnAfterRateObject            func(packet nex.PacketInterface, target datastore_types.DataStoreRatingTarget, param datastore_types.DataStoreRateObjectParam, fetchRatings types.Bool)
	OnAfterPostMetaBinary        func(packet nex.PacketInterface, param datastore_types.DataStorePreparePostParam)
	OnAfterPreparePostObject     func(packet nex.PacketInterface, param datastore_types.DataStorePreparePostParam)
	OnAfterPrepareGetObject      func(packet nex.PacketInterface, param datastore_types.DataStorePrepareGetParam)
	OnAfterCompletePostObject    func(packet nex.PacketInterface, param datastore_types.DataStoreCompletePostParam)
	OnAfterGetMetasMultipleParam func(packet nex.PacketInterface, params types.List[datastore_types.DataStoreGetMetaParam])
	OnAfterCompletePostObjects   func(packet nex.PacketInterface, dataIDs types.List[types.UInt64])
	OnAfterChangeMeta            func(packet nex.PacketInterface, param datastore_types.DataStoreChangeMetaParam)
	OnAfterRateObjects           func(packet nex.PacketInterface, targets types.List[datastore_types.DataStoreRatingTarget], params types.List[datastore_types.DataStoreRateObjectParam], transactional types.Bool, fetchRatings types.Bool)
}

// SetManager defines the matchmaking manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.DataStoreManager) {
	var err error

	commonProtocol.manager = manager

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS datastore`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * User uploaded data starts at DataID 1,000,000.
	// * The server, however, can upload "system" data
	// * in values 900,000 to 999,999. We only need to
	// * worry about user generated data here though
	_, err = manager.Database.Exec(`CREATE SEQUENCE IF NOT EXISTS datastore.object_data_id_seq
		INCREMENT 1
		START 1000000`,
	)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * DataStore only updates an objects "update time" in specific instances.
	// * To account for this, `update_date` refers to the time expected by NEX.
	// * `true_update_date` tracks the TRUE update time, noting the most recent
	// * update to any metadata regardless of what NEX expects
	// *
	// * Additionally, later versions of NEX seem to have disabled the "reference"
	// * features. Most modern games (3.5+?) seem to no longer track the reference
	// * counts or last reference dates, with some exceptions such as AC:NL. We
	// * have opted to track them internally anyway, regardless of NEX version. We
	// * may add a flag in the future for disabling the *sending* of these results
	// * by just setting them all to 0 at response time, for accuracy sake
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.objects (
		data_id numeric(20) NOT NULL DEFAULT nextval('datastore.object_data_id_seq') PRIMARY KEY CHECK (data_id BETWEEN 0 AND 18446744073709551615),
		deleted boolean NOT NULL DEFAULT FALSE,
		owner numeric(20) CHECK (owner BETWEEN 0 AND 18446744073709551615), -- Wii U/3DS clients only need a uint32, but the Switch uses uint64 PIDs

		-- Data set in DataStorePreparePostParam/DataStorePreparePostParamV1
		name text,
		data_type int CHECK (data_type BETWEEN 0 AND 65535),
		meta_binary bytea,
		access_permission smallint CHECK (access_permission BETWEEN 0 AND 4),
		access_permission_recipients numeric(20)[] CHECK (0 <= ALL(access_permission_recipients) AND 18446744073709551615 >= ALL(access_permission_recipients)),
		update_permission smallint CHECK (update_permission BETWEEN 0 AND 4),
		update_permission_recipients numeric(20)[] CHECK (0 <= ALL(update_permission_recipients) AND 18446744073709551615 >= ALL(update_permission_recipients)),
		raw_flags numeric(10) CHECK (raw_flags BETWEEN 0 AND 4294967295),
		expiration_days int CHECK (expiration_days BETWEEN 0 AND 365),
		refer_data_id numeric(10) CHECK (refer_data_id BETWEEN 0 AND 4294967295), -- this is another data_id, but it can ONLY use the uint32 space
		tags text[],
		persistence_slot_id int CHECK (persistence_slot_id BETWEEN 0 AND 15 OR persistence_slot_id = 65535),

		-- Decoded raw_flags
		-- Only supports stock flags, custom flags must be handled separately
		needs_review boolean NOT NULL DEFAULT FALSE,
		update_expiration_on_reference boolean NOT NULL DEFAULT FALSE,
		use_read_lock boolean NOT NULL DEFAULT FALSE,
		notify_access_recipients_on_creation boolean NOT NULL DEFAULT FALSE,
		notify_access_recipients_on_update boolean NOT NULL DEFAULT FALSE,
		not_use_file_server boolean NOT NULL DEFAULT FALSE,
		need_upload_completion boolean NOT NULL DEFAULT FALSE,

		-- System/internal fields
		lock_id numeric(10) CHECK (lock_id BETWEEN 0 AND 4294967295),
		status smallint CHECK (status BETWEEN 0 AND 5),
		access_password numeric(20) CHECK (access_password BETWEEN 0 AND 18446744073709551615),
		update_password numeric(20) CHECK (update_password BETWEEN 0 AND 18446744073709551615),
		reference_count numeric(10) NOT NULL DEFAULT 0 CHECK (reference_count BETWEEN 0 AND 4294967295),
		creation_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		update_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		true_update_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		last_reference_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		expiration_date timestamp
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * Versions as created through PrepareUpdateObject, including version 1 which is
	// * the initial version of the object when it got posted. Objects can have more than
	// * one "pending" versions (updates) by calling PrepareUpdateObject more than once
	// * without completing them. Each of these pending versions is valid, and can be
	// * completed independently, so each needs to be tracked separately
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.object_versions (
		data_id numeric(20) REFERENCES datastore.objects(data_id),
		version numeric(10) CHECK (version BETWEEN 0 AND 4294967295),
		size numeric(10) CHECK (size BETWEEN 0 AND 4294967295),
		extra_data text[],
		preparer numeric(20) CHECK (preparer BETWEEN 0 AND 18446744073709551615),
		completed boolean NOT NULL DEFAULT FALSE,
		creation_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		PRIMARY KEY (data_id, version)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * Object persistence is handled at the user level,
	// * not the object level
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.persistence_slots (
		pid numeric(20) CHECK (pid BETWEEN 0 AND 18446744073709551615),
		slot int CHECK (slot BETWEEN 0 AND 15),
		data_id numeric(20) REFERENCES datastore.objects(data_id),
		PRIMARY KEY (pid, slot)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.rating_slots (
		data_id numeric(20) NOT NULL REFERENCES datastore.objects(data_id),
		total_value numeric(20) NOT NULL CHECK (total_value BETWEEN -9223372036854775808 AND 9223372036854775807),
		rating_count numeric(10) NOT NULL DEFAULT 0 CHECK (rating_count BETWEEN 0 AND 4294967295),

		-- Data set in DataStoreRatingInitParamWithSlot
		slot smallint CHECK (slot BETWEEN 0 AND 15),
		raw_flags smallint CHECK (raw_flags BETWEEN 0 AND 255),
		raw_internal_flags smallint CHECK (raw_internal_flags BETWEEN 0 AND 255),
		minimum_value numeric(10) CHECK (minimum_value BETWEEN -2147483648 AND 2147483647),
		maximum_value numeric(10) CHECK (maximum_value BETWEEN -2147483648 AND 2147483647),
		initial_value numeric(20) CHECK (initial_value BETWEEN -9223372036854775808 AND 9223372036854775807),
		lock_type smallint CHECK (lock_type BETWEEN 0 AND 3),
		lock_period_duration int CHECK (lock_period_duration BETWEEN -32768 AND 32767),
		lock_period_hour smallint CHECK (lock_period_hour BETWEEN -128 AND 127),

		-- Decoded raw_flags
		-- Only supports stock flags, custom flags must be handled separately
		allow_multiple_ratings boolean NOT NULL DEFAULT FALSE,
		round_negatives boolean NOT NULL DEFAULT TRUE,
		disable_self_rating boolean NOT NULL DEFAULT FALSE,

		-- Decoded raw_internal_flags
		-- Only supports stock flags, custom flags must be handled separately
		use_minimum boolean NOT NULL DEFAULT FALSE,
		use_maximum boolean NOT NULL DEFAULT FALSE,

		PRIMARY KEY (data_id, slot)
	);`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * The rating log as used by NEX. For actual logs, see rating_history
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.rating_logs (
		data_id numeric(20),
		slot smallint,
		pid numeric(20) CHECK (pid BETWEEN 0 AND 18446744073709551615),
		value int,
		locked_until timestamp,
		created_at timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		updated_at timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		PRIMARY KEY (data_id, slot, pid),
		FOREIGN KEY (data_id, slot) REFERENCES datastore.rating_slots(data_id, slot)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * Every rating created for every slot in the order the ratings were created. This is used
	// * as an internal history/log, not used by the games, so we can audit hacked ratings and
	// * remove them if need be. If a rating needs to be removed, then the total_value and rating_count
	// * values need to be recomputed
	// TODO - Use this log to compute total_value and rating_count on the fly? Modifiable ratings with rounded totals make this special. From the wiki: "For example, if a user rates -10 (the total is rounded to 0), another user rates 3 (the total is 3), and the first user changes their rating to 1, the total becomes <code>3 - (-10) + 1 = 14</code>"
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.rating_history (
		id bigserial PRIMARY KEY,
		data_id numeric(20) NOT NULL,
		slot smallint NOT NULL,
		pid numeric(20) NOT NULL CHECK (pid BETWEEN 0 AND 18446744073709551615),
		value int NOT NULL,
		removed boolean NOT NULL DEFAULT FALSE,
		created_at timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		FOREIGN KEY (data_id, slot) REFERENCES datastore.rating_slots(data_id, slot)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE INDEX IF NOT EXISTS rating_history_slot_pid_idx ON datastore.rating_history (data_id, slot, pid)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE INDEX IF NOT EXISTS rating_history_pid_idx ON datastore.rating_history (pid)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE SEQUENCE IF NOT EXISTS datastore.notification_id_seq`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.notifications (
		id numeric(20) NOT NULL DEFAULT nextval('datastore.notification_id_seq') PRIMARY KEY CHECK (id BETWEEN 0 AND 18446744073709551615),
		read boolean NOT NULL DEFAULT FALSE,
		recipient_id numeric(20) CHECK (recipient_id BETWEEN 0 AND 18446744073709551615),
		data_id numeric(20) CHECK (data_id BETWEEN 0 AND 18446744073709551615),

		-- The following fields are used for tracking purposes
		sender_pid numeric(20) CHECK (sender_pid BETWEEN 0 AND 18446744073709551615),
		creation_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		read_date timestamp,

		-- A user only ever has one notification for each object
		UNIQUE (recipient_id, data_id)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol datastore.Interface) *CommonProtocol {
	endpoint := protocol.Endpoint()

	commonProtocol := &CommonProtocol{
		endpoint: endpoint,
		protocol: protocol,
	}

	// TODO - Reorder this by method ID
	protocol.SetHandlerGetMeta(commonProtocol.getMeta)
	protocol.SetHandlerGetMetas(commonProtocol.getMetas)
	protocol.SetHandlerGetMetasMultipleParam(commonProtocol.getMetasMultipleParam)
	protocol.SetHandlerPreparePostObject(commonProtocol.preparePostObject)
	protocol.SetHandlerPrepareGetObject(commonProtocol.prepareGetObject)
	protocol.SetHandlerCompletePostObject(commonProtocol.completePostObject)
	protocol.SetHandlerChangeMeta(commonProtocol.changeMeta)
	protocol.SetHandlerGetPasswordInfo(commonProtocol.getPasswordInfo)
	protocol.SetHandlerGetPasswordInfos(commonProtocol.getPasswordInfos)
	protocol.SetHandlerCompletePostObjects(commonProtocol.completePostObjects)
	protocol.SetHandlerDeleteObject(commonProtocol.deleteObject)
	protocol.SetHandlerDeleteObjects(commonProtocol.deleteObjects)
	protocol.SetHandlerPostMetaBinary(commonProtocol.postMetaBinary)
	protocol.SetHandlerPrepareGetObjectV1(commonProtocol.prepareGetObjectV1)
	protocol.SetHandlerPreparePostObjectV1(commonProtocol.preparePostObjectV1)
	protocol.SetHandlerCompletePostObjectV1(commonProtocol.completePostObjectV1)
	protocol.SetHandlerChangeMetaV1(commonProtocol.changeMetaV1)
	protocol.SetHandlerChangeMetasV1(commonProtocol.changeMetasV1)
	protocol.SetHandlerChangeMetas(commonProtocol.changeMetas)
	protocol.SetHandlerGetSpecificMeta(commonProtocol.getSpecificMeta)
	protocol.SetHandlerGetSpecificMetaV1(commonProtocol.getSpecificMetaV1)
	protocol.SetHandlerTouchObject(commonProtocol.touchObject)
	protocol.SetHandlerPostMetaBinaryWithDataID(commonProtocol.stubPostMetaBinaryWithDataID)
	protocol.SetHandlerPostMetaBinariesWithDataID(commonProtocol.stubPostMetaBinariesWithDataID)
	protocol.SetHandlerPrepareUpdateObject(commonProtocol.prepareUpdateObject)
	protocol.SetHandlerCompleteUpdateObject(commonProtocol.completeUpdateObject)
	protocol.SetHandlerGetPersistenceInfo(commonProtocol.getPersistenceInfo)
	protocol.SetHandlerGetPersistenceInfos(commonProtocol.getPersistenceInfos)
	protocol.SetHandlerPerpetuateObject(commonProtocol.perpetuateObject)
	protocol.SetHandlerUnperpetuateObject(commonProtocol.unperpetuateObject)
	protocol.SetHandlerPrepareGetObjectOrMetaBinary(commonProtocol.prepareGetObjectOrMetaBinary)
	protocol.SetHandlerGetObjectInfos(commonProtocol.getObjectInfos)
	protocol.SetHandlerRateObject(commonProtocol.rateObject)
	protocol.SetHandlerRateObjects(commonProtocol.rateObjects)
	protocol.SetHandlerRateObjectWithPosting(commonProtocol.rateObjectWithPosting)
	protocol.SetHandlerRateObjectsWithPosting(commonProtocol.rateObjectsWithPosting)
	protocol.SetHandlerGetRating(commonProtocol.getRating)
	protocol.SetHandlerGetRatings(commonProtocol.getRatings)
	protocol.SetHandlerGetRatingWithLog(commonProtocol.getRatingWithLog)
	protocol.SetHandlerResetRating(commonProtocol.resetRating)
	protocol.SetHandlerResetRatings(commonProtocol.resetRatings)
	protocol.SetHandlerGetNotificationURL(commonProtocol.getNotificationURL)
	protocol.SetHandlerGetNewArrivedNotifications(commonProtocol.getNewArrivedNotfications)
	protocol.SetHandlerGetNewArrivedNotificationsV1(commonProtocol.getNewArrivedNotficationsV1)

	return commonProtocol
}
