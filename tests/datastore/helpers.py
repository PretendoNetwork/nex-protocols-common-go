import contextlib

import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants

UNKNOWN_DATA_ID = 0xFFFFFF
UINT32_MAX = 0xFFFFFFFF
UINT64_MAX = 0xFFFFFFFFFFFFFFFF

# * Object strings go on the wire as UTF-8, but the server decodes them
# * into 16 bit characters, so everything in the basic multilingual plane
# * survives and nothing outside it does. These all come back unchanged
TEXT_SAMPLES = [
	pytest.param("hello world", id="ascii"),
	pytest.param("caf\u00e9 na\u00efve", id="latin-1-accents"),
	pytest.param("stra\u00dfe", id="the-german-sharp-s"),
	pytest.param("\u03b1\u03b2\u03b3", id="greek"),
	pytest.param("\u043f\u0440\u0438\u0432\u0435\u0442", id="cyrillic"),
	pytest.param("\u30c6\u30b9\u30c8", id="japanese-kana"),
	pytest.param("\u6f22\u5b57", id="japanese-kanji"),
	pytest.param("\ud55c\uae00", id="korean"),
	pytest.param("\u0645\u0631\u062d\u0628\u0627", id="right-to-left"),
	pytest.param("e\u0301", id="a-combining-accent"),
	pytest.param("\uff8a\uff9e", id="halfwidth-katakana"),
	pytest.param("\uff28\uff25\uff2c\uff2c\uff2f", id="fullwidth-latin"),
	pytest.param("\ufffd", id="the-replacement-character"),
	pytest.param("   ", id="only-spaces"),
]

# * Anything above the basic multilingual plane needs a surrogate pair,
# * which the server cannot decode. Real server throws PythonCore::ConversionError
# * which we'd need to emulate?
ASTRAL_TEXT_SAMPLES = [
	pytest.param("\U0001F3AE", id="an-emoji"),
	pytest.param("\U0001F3AE\U0001F47E", id="two-emoji"),
	pytest.param("\U0001F468\u200D\U0001F4BB", id="emoji-joined-by-a-zero-width-joiner"),
]

# * For some reason object NAMES accept these, but object TAGS
# * throw DataStore::InvalidArgument?
CONTROL_TEXT_SAMPLES = [
	pytest.param("a\tb", id="a-tab"),
	pytest.param("a\nb", id="a-newline"),
	pytest.param("a\rb", id="a-carriage-return"),
]

@contextlib.asynccontextmanager
async def connect(pid: int):
	"""
	Connects a user by PID to the test server, as a DataStore client
	"""

	async with nex.connect(pid) as client:
		yield datastore.DataStoreClient(client)

def permission(**fields) -> datastore.DataStorePermission:
	"""
	Creates a DataStorePermission with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePermission()
	param.permission = constants.Permission.PERMISSION_PUBLIC
	param.recipients = []

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def post_param(**fields) -> datastore.DataStorePreparePostParam:
	"""
	Creates a DataStorePreparePostParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePreparePostParam()
	param.size = 0
	param.name = "test"
	param.data_type = 0
	param.meta_binary = b"meta"
	param.permission = permission()
	param.delete_permission = permission(permission=constants.Permission.PERMISSION_PRIVATE)
	param.flag = constants.DataFlag.DATA_FLAG_NONE
	param.period = constants.DEFAULT_PERIOD
	param.extra_data = ["WUP", "4", "EUR", "110", "GB", ""]

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def post_param_v1(**fields) -> datastore.DataStorePreparePostParamV1:
	"""
	Creates a DataStorePreparePostParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePreparePostParamV1()
	param.size = 0
	param.name = "test"
	param.data_type = 0
	param.meta_binary = b"meta"
	param.permission = permission()
	param.delete_permission = permission(permission=constants.Permission.PERMISSION_PRIVATE)
	param.flag = constants.DataFlag.DATA_FLAG_NONE
	param.period = constants.DEFAULT_PERIOD
	param.refer_data_id = constants.INVALID_DATAID
	param.tags = []
	param.rating_init_param = []

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def delete_param(**fields) -> datastore.DataStoreDeleteParam:
	"""
	Creates a DataStoreDeleteParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreDeleteParam()
	param.data_id = constants.INVALID_DATAID
	param.update_password = constants.INVALID_PASSWORD

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def rating_init_param(**fields) -> datastore.DataStoreRatingInitParam:
	"""
	Creates a DataStoreRatingInitParam with default parameters, using
	no locks or limits. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreRatingInitParam()
	param.flag = 0
	param.internal_flag = 0
	param.lock_type = constants.RatingLockType.RATING_LOCK_NONE
	param.initial_value = 0
	param.range_min = 0
	param.range_max = 0
	param.period_hour = 0
	param.period_duration = 0

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def rating_init_param_with_slot(**fields) -> datastore.DataStoreRatingInitParamWithSlot:
	"""
	Creates a DataStoreRatingInitParamWithSlot with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreRatingInitParamWithSlot()
	param.slot = 0
	param.param = rating_init_param()

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def persistence_init_param(**fields) -> datastore.DataStorePersistenceInitParam:
	"""
	Creates a DataStorePersistenceInitParam with default parameters, which
	does not persist the object. Any field can be overridden with a keyword
	argument
	"""

	param = datastore.DataStorePersistenceInitParam()
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID
	param.delete_last_object = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def persistence_target(**fields) -> datastore.DataStorePersistenceTarget:
	"""
	Creates a DataStorePersistenceTarget with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePersistenceTarget()
	param.owner_id = 0
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def complete_post_param(**fields) -> datastore.DataStoreCompletePostParam:
	"""
	Creates a DataStoreCompletePostParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParam()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def change_meta_compare_param(**fields) -> datastore.DataStoreChangeMetaCompareParam:
	"""
	Creates a DataStoreChangeMetaCompareParam with default parameters,
	which compares nothing. Any field can be overridden with a keyword
	argument
	"""

	param = datastore.DataStoreChangeMetaCompareParam()
	param.comparison_flag = constants.ComparisonFlag.COMPARISON_FLAG_NONE
	param.name = ""
	param.permission = permission()
	param.delete_permission = permission()
	param.period = constants.DEFAULT_PERIOD
	param.meta_binary = b""
	param.tags = []
	param.referred_count = 0
	param.data_type = 0
	param.status = constants.DataStatus.DATA_STATUS_NONE

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def change_meta_param(**fields) -> datastore.DataStoreChangeMetaParam:
	"""
	Creates a DataStoreChangeMetaParam with default parameters, which
	changes nothing. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreChangeMetaParam()
	param.data_id = constants.INVALID_DATAID
	param.modifies_flag = constants.ModificationFlag.MODIFICATION_FLAG_NONE
	param.name = ""
	param.permission = permission()
	param.delete_permission = permission()
	param.period = constants.DEFAULT_PERIOD
	param.meta_binary = b""
	param.tags = []
	param.update_password = constants.INVALID_PASSWORD
	param.referred_count = 0
	param.data_type = 0
	param.status = constants.DataStatus.DATA_STATUS_NONE
	param.compare_param = change_meta_compare_param()
	param.persistence_target = persistence_target()

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def complete_post_param_v1(**fields) -> datastore.DataStoreCompletePostParamV1:
	"""
	Creates a DataStoreCompletePostParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParamV1()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def touch_object_param(**fields) -> datastore.DataStoreTouchObjectParam:
	"""
	Creates a DataStoreTouchObjectParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreTouchObjectParam()
	param.data_id = constants.INVALID_DATAID
	param.lock_id = 0
	param.access_password = constants.INVALID_PASSWORD

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def prepare_get_param(**fields) -> datastore.DataStorePrepareGetParam:
	"""
	Creates a DataStorePrepareGetParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePrepareGetParam()
	param.data_id = constants.INVALID_DATAID
	param.lock_id = 0
	param.persistence_target = persistence_target()
	param.access_password = constants.INVALID_PASSWORD
	param.extra_data = []

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def prepare_get_param_v1(**fields) -> datastore.DataStorePrepareGetParamV1:
	"""
	Creates a DataStorePrepareGetParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePrepareGetParamV1()
	param.data_id = constants.INVALID_DATAID
	param.lock_id = 0

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def get_meta_param(**fields) -> datastore.DataStoreGetMetaParam:
	"""
	Creates a DataStoreGetMetaParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreGetMetaParam()
	param.data_id = constants.INVALID_DATAID
	param.persistence_target = persistence_target()
	param.result_option = 0
	param.access_password = 0

	for name, value in fields.items():
		setattr(param, name, value)

	return param

async def post_meta_binary(pid: int, **fields) -> int:
	"""
	Posts an object which does not use the file server as the
	given user, returning its data ID
	"""

	async with connect(pid) as client:
		return await client.post_meta_binary(post_param(**fields))

async def prepare_post_object(pid: int, **fields) -> int:
	"""
	Starts posting an object which uses the file server, as the given
	user, returning its data ID. The upload is not completed
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object(post_param(**fields))

		return post_info.data_id

async def prepare_post_object_v1(pid: int, **fields) -> int:
	"""
	Starts posting an object which uses the file server, as the given user,
	using the older version of the method. The upload is not completed
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object_v1(post_param_v1(**fields))

		return post_info.data_id

async def delete_object(pid: int, **fields) -> None:
	"""
	Deletes an object as the given user
	"""

	async with connect(pid) as client:
		await client.delete_object(delete_param(**fields))

async def post_object(pid: int, **fields) -> int:
	"""
	Posts an object which uses the file server as the given user, returning
	its data ID. The object contents is never uploaded
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object(post_param(**fields))
		await client.complete_post_object(complete_post_param(data_id=post_info.data_id))

		return post_info.data_id

async def post_object_v1(pid: int, **fields) -> int:
	"""
	Posts an object which uses the file server as the given user, using the
	older version of the method, returning its data ID. The object contents
	is never uploaded
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object_v1(post_param_v1(**fields))
		await client.complete_post_object_v1(complete_post_param_v1(data_id=post_info.data_id))

		return post_info.data_id

async def change_meta(pid: int, **fields) -> None:
	"""
	Changes an objects metadata as the given user
	"""

	async with connect(pid) as client:
		await client.change_meta(change_meta_param(**fields))

async def get_password_info(pid: int, data_id: int) -> datastore.DataStorePasswordInfo:
	"""
	Gets an objects passwords as the given user. Only the object
	owner is allowed to do this
	"""

	async with connect(pid) as client:
		return await client.get_password_info(data_id)

async def touch_object(pid: int, **fields) -> None:
	"""
	Touches an object as the given user
	"""

	async with connect(pid) as client:
		await client.touch_object(touch_object_param(**fields))

async def get_persistence_info(pid: int, owner_id: int, slot: int) -> datastore.DataStorePersistenceInfo:
	"""
	Reads one of a users persistence slots, as the given user
	"""

	async with connect(pid) as client:
		return await client.get_persistence_info(owner_id, slot)

async def get_persistence_info_error(pid: int, owner_id: int, slot: int) -> str:
	"""
	Runs DataStoreProtocol::GetPersistenceInfo and expects an error to
	be thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with connect(pid) as client:
		try:
			await client.get_persistence_info(owner_id, slot)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetPersistenceInfo did not throw an error")

async def get_meta(pid: int, **fields) -> datastore.DataStoreMetaInfo:
	"""
	Gets an objects metadata as the given user
	"""

	async with connect(pid) as client:
		return await client.get_meta(get_meta_param(**fields))

async def get_meta_error(pid: int, **fields) -> str:
	"""
	Runs DataStoreProtocol::GetMeta and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with connect(pid) as client:
		try:
			await client.get_meta(get_meta_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetMeta did not throw an error")

async def post_persisted(post, pid: int, slot: int, **fields) -> int:
	"""
	Posts an object into one of the users persistence slots,
	using the given post method, and returns its data ID
	"""

	init_param = persistence_init_param(persistence_id=slot)

	return await post(pid, persistence_init_param=init_param, **fields)

# * Objects can be posted with or without the file server. Tests which
# * upload one are run against both, to make sure the method being
# * tested behaves the same either way
POST_METHODS = [
	pytest.param(post_meta_binary, id="post-meta-binary"),
	pytest.param(post_object, id="post-object"),
]

# * Used to ensure that tests involving objects which use the file server
# * like PrepareGetObject, are ran against objects posted through both
# * ways of creating objects on the file server
FILE_SERVER_POST_METHODS = [
	pytest.param(post_object, id="post-object"),
	pytest.param(post_object_v1, id="post-object-v1"),
]

def result_names(results: list) -> list:
	"""
	Turns the results of a batch request into something readable, where
	an entry which worked is None
	"""

	return [None if result.is_success() else result.name() for result in results]

def data_ids_of(response) -> list[int]:
	"""
	The data ID of every info which came back
	"""

	return [info.data_id for info in response.infos]
