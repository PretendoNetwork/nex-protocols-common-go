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

	// TODO - Enforce size limits at the database level in all tables? Or keep it at the application level?

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
	// TODO - Store every object version as it's own row, or keep a single row?
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.objects (
		data_id numeric(20) NOT NULL DEFAULT nextval('datastore.object_data_id_seq') PRIMARY KEY,
		version numeric(10) NOT NULL DEFAULT 1,
		deleted boolean NOT NULL DEFAULT FALSE,
		owner numeric(20), -- Wii U/3DS clients only need a uint32, but the Switch uses uint64 PIDs

		-- Data set in DataStorePreparePostParam/DataStorePreparePostParamV1
		size numeric(10),
		name text,
		data_type int,
		meta_binary bytea,
		access_permission smallint,
		access_permission_recipients numeric(20)[],
		update_permission smallint,
		update_permission_recipients numeric(20)[],
		raw_flags numeric(10),
		expiration_days int, -- this can only be between 0-365, but is sent as a uint16
		refer_data_id numeric(10), -- this is another data_id, but it can ONLY use the uint32 space
		tags text[],
		persistence_slot_id int,
		extra_data text[],

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
		lock_id numeric(10),
		status smallint, -- this can only be between 1-5, but allocate enough space anyway
		access_password numeric(20),
		update_password numeric(20),
		reference_count numeric(10) NOT NULL DEFAULT 0,
		creation_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		update_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		true_update_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		last_reference_date timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		expiration_date timestamp,
		upload_completed boolean
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * Object persistence is handled at the user level,
	// * not the object level
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.persistence_slots (
		pid numeric(20),
		slot int, -- can technically only be 0-15, but sent as a uint16
		data_id numeric(20) REFERENCES datastore.objects(data_id),
		PRIMARY KEY (pid, slot)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.rating_settings (
		data_id numeric(20) NOT NULL REFERENCES datastore.objects(data_id),

		-- Data set in DataStoreRatingInitParamWithSlot
		slot smallint, -- can technically only be 0-15, but allocate enough space anyway
		raw_flags smallint,
		raw_internal_flags smallint,
		minimum_value numeric(10),
		maximum_value numeric(10),
		initial_value numeric(20),
		lock_type smallint, -- can technically only be 0-3, but allocate enough space anyway
		lock_period_duration int,
		lock_period_hour smallint, -- can technically only hold 2 digits, but allocate enough space anyway

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

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.ratings (
		id serial PRIMARY KEY,
		data_id numeric(20),
		slot smallint,
		pid numeric(20),
		value numeric(20),
		created_at timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		updated_at timestamp DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'UTC'),
		FOREIGN KEY (data_id, slot) REFERENCES datastore.rating_settings(data_id, slot)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS datastore.rating_locks (
		pid numeric(20),
		data_id numeric(20),
		slot smallint,
		locked_until timestamp,
		PRIMARY KEY (pid, data_id, slot),
		FOREIGN KEY (data_id, slot) REFERENCES datastore.rating_settings(data_id, slot)
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
	protocol.SetHandlerGetObjectInfos(commonProtocol.getObjectInfos) // TODO - Fix this methods type in protocols lib
	protocol.SetHandlerRateObject(commonProtocol.rateObject)
	protocol.SetHandlerRateObjects(commonProtocol.rateObjects)
	protocol.SetHandlerGetRating(commonProtocol.getRating)
	protocol.SetHandlerGetRatings(commonProtocol.getRatings)
	protocol.SetHandlerGetRatingWithLog(commonProtocol.getRatingWithLog)
	protocol.SetHandlerResetRating(commonProtocol.resetRating)
	protocol.SetHandlerResetRatings(commonProtocol.resetRatings) // TODO - Fix this methods type in protocols lib

	return commonProtocol
}
