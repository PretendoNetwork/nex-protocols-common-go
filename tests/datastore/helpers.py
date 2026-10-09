import contextlib

import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants

UNKNOWN_DATA_ID = 0xFFFFFF
UINT32_MAX = 0xFFFFFFFF
UINT64_MAX = 0xFFFFFFFFFFFFFFFF
INT32_MAX = 0x7FFFFFFF
INT32_MIN = -0x80000000
INT64_MAX = 0x7FFFFFFFFFFFFFFF
INT64_MIN = -0x8000000000000000

# * The largest object which can be uploaded
# TODO - Make a constant for this in Go libs
MAX_OBJECT_SIZE = 10 * 1024 * 1024

# * Object strings go on the wire as UTF-8, but the server decodes them
# * into 16 bit characters, so everything in the basic multilingual plane
# * survives and nothing outside it does. These all come back unchanged
TEXT_SAMPLES = [
	pytest.param("hello world", id="ascii"),
	pytest.param("caf\u00e9 na\u00efve", id="latin-1-accents"),
	pytest.param("\u6f22\u5b57", id="japanese-kanji"),
	pytest.param("e\u0301", id="a-combining-accent"),
	pytest.param("\uff28\uff25\uff2c\uff2c\uff2f", id="fullwidth-latin"),
	pytest.param("   ", id="only-spaces")
]

# * Anything above the basic multilingual plane needs a surrogate pair,
# * which the server cannot decode. Real server throws PythonCore::ConversionError
# * which we'd need to emulate?
ASTRAL_TEXT_SAMPLES = [
	pytest.param("\U0001F3AE", id="an-emoji")
]

# * For some reason object NAMES accept these, but object TAGS
# * throw DataStore::InvalidArgument?
CONTROL_TEXT_SAMPLES = [
	pytest.param("a\tb", id="a-tab"),
	pytest.param("a\nb", id="a-newline"),
	pytest.param("a\rb", id="a-carriage-return")
]

# * Every permission, with a recipient for the ones which use them
PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], id="specified"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], id="private"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], id="specified-friend")
]

# * Every access permission, with who reads an object using it and whether
# * they can see it. Objects are always visible to their owner
ACCESS_PERMISSION_CASES = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.FRIEND_A, True, id="public-owner"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.FRIEND_B, True, id="public-friend"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, True, id="public-stranger"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_A, True, id="friend-owner"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, True, id="friend-friend"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.STRANGER, False, id="friend-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.FRIEND_A, True, id="specified-owner"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, True, id="specified-recipient"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.FRIEND_B, False, id="specified-not-recipient"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.FRIEND_A, True, id="private-owner"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.FRIEND_B, False, id="private-friend"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.STRANGER, False, id="private-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.FRIEND_A, True, id="specified-friend-owner"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.FRIEND_B, True, id="specified-friend-recipient-friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.STRANGER, False, id="specified-friend-recipient-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], nex.FRIEND_B, False, id="specified-friend-friend-not-recipient")
]

# * Each update permission, who tries to update or reset with it, and whether
# * they may. The owner always may
UPDATE_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.STRANGER, False, id="private"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, True, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, True, id="friend-to-a-friend"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.STRANGER, False, id="friend-to-a-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, True, id="specified-to-a-recipient"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, True, id="specified-friend-to-a-friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], nex.STRANGER, False, id="specified-friend-to-a-stranger")
]

# * Every access permission, with who is notified when an object using it is
# * posted with DATA_FLAG_USE_NOTIFICATION_ON_POST
NOTIFIED_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], [], id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], [nex.FRIEND_B], id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], [nex.STRANGER], id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.FRIEND_A, nex.STRANGER], [nex.FRIEND_A, nex.STRANGER], id="specified-with-the-owner"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], [], id="private"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], [nex.FRIEND_B], id="specified-friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], [], id="specified-friend-to-a-stranger")
]

# * The persistence slots tests use. Every test which needs a slot uses these,
# * and nothing is ever put in EMPTY_PERSISTENCE_SLOT
PERSISTENCE_SLOT = 0
SECOND_PERSISTENCE_SLOT = 1
EMPTY_PERSISTENCE_SLOT = 15

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

# * The size objects are posted with, when a test needs to see the size change
POSTED_SIZE = 10

# * The size objects are updated to, different from POSTED_SIZE
UPDATED_SIZE = 20

# * How many objects to post when looking at entropy
PASSWORD_SAMPLES = 16

# * The smallest gap allowed between two sorted passwords. A good
# * generator leaves huge gaps between passwords. This tries to
# * catch sequential generators and generators which produce small
# * gaps, but is intentionally somewhat small as to not trip any
# * false negatives
CLOSEST_SENSIBLE_PASSWORDS = 1000000

# * Metadata an object is posted with, so that a change to it can be seen
BEFORE_NAME = "before"
BEFORE_META_BINARY = b"BEFORE"
BEFORE_TAGS = ["before"]
BEFORE_PERIOD = 90

# * The comparison flag for each field of a DataStoreChangeMetaCompareParam
COMPARISON_FLAGS = {
	"name": constants.ComparisonFlag.COMPARISON_FLAG_NAME,
	"permission": constants.ComparisonFlag.COMPARISON_FLAG_ACCESS_PERMISSION,
	"delete_permission": constants.ComparisonFlag.COMPARISON_FLAG_UPDATE_PERMISSION,
	"period": constants.ComparisonFlag.COMPARISON_FLAG_PERIOD,
	"meta_binary": constants.ComparisonFlag.COMPARISON_FLAG_METABINARY,
	"tags": constants.ComparisonFlag.COMPARISON_FLAG_TAGS,
	"referred_count": constants.ComparisonFlag.COMPARISON_FLAG_REFERRED_COUNT,
	"data_type": constants.ComparisonFlag.COMPARISON_FLAG_DATA_TYPE,
	"status": constants.ComparisonFlag.COMPARISON_FLAG_STATUS
}

# * For some reason these flags are accepted even though they have no field?
FLAGS_WITH_NO_FIELD = [
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, id="referred-count"),
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_STATUS, id="status")
]

# * Every rating lock type
RATING_LOCK_TYPES = [
	pytest.param(constants.RatingLockType.RATING_LOCK_NONE, id="none"),
	pytest.param(constants.RatingLockType.RATING_LOCK_INTERVAL, id="interval"),
	pytest.param(constants.RatingLockType.RATING_LOCK_PERIOD, id="period"),
	pytest.param(constants.RatingLockType.RATING_LOCK_PERMANENT, id="permanent")
]

# * Every stock flag, and whether an object on the file server can be posted
# * with it. Only DATA_FLAG_NOT_USE_FILESERVER cannot be used, as it's set by
# * the game server in PostMetaBinary
FILE_SERVER_DATA_FLAGS = [
	pytest.param(constants.DataFlag.DATA_FLAG_NEED_REVIEW, True, id="need-review"),
	pytest.param(constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED, True, id="period-from-last-referred"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_READ_LOCK, True, id="use-read-lock"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST, True, id="use-notification-on-post"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE, True, id="use-notification-on-update"),
	pytest.param(constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER, False, id="not-use-fileserver"),
	pytest.param(constants.DataFlag.DATA_FLAG_NEED_COMPLETION, True, id="need-completion")
]

# * Every access permission which lets someone other than the owner in, with
# * a user it lets in
ACCESS_PERMISSION_VIEWERS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, id="specified-friend")
]

# * Every access permission which lets someone other than the owner in, with
# * a user it lets in and a user it keeps out. A public object keeps nobody
# * out, so it has no user kept out
ACCESS_PERMISSION_VIEWERS_AND_OUTSIDERS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, None, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, nex.STRANGER, id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, nex.FRIEND_B, id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, nex.STRANGER, id="specified-friend")
]

@contextlib.asynccontextmanager
async def connect(pid: int):
	"""
	Connects a user by PID to the test server, as a DataStore client
	"""

	async with nex.connect(pid) as client:
		yield datastore.DataStoreClient(client)

def override(param, fields: dict):
	"""
	Overrides fields on a param, checking for invalid ones
	"""

	for name, value in fields.items():
		if not hasattr(param, name):
			raise AttributeError("%s has no field %r" % (type(param).__name__, name))

		setattr(param, name, value)

	return param

def permission(**fields) -> datastore.DataStorePermission:
	"""
	Creates a DataStorePermission with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePermission()
	param.permission = constants.Permission.PERMISSION_PUBLIC
	param.recipients = []

	return override(param, fields)

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

	return override(param, fields)

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

	return override(param, fields)

def delete_param(**fields) -> datastore.DataStoreDeleteParam:
	"""
	Creates a DataStoreDeleteParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreDeleteParam()
	param.data_id = constants.INVALID_DATAID
	param.update_password = constants.INVALID_PASSWORD

	return override(param, fields)

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

	return override(param, fields)

def rating_init_param_with_slot(**fields) -> datastore.DataStoreRatingInitParamWithSlot:
	"""
	Creates a DataStoreRatingInitParamWithSlot with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreRatingInitParamWithSlot()
	param.slot = 0
	param.param = rating_init_param()

	return override(param, fields)

def persistence_init_param(**fields) -> datastore.DataStorePersistenceInitParam:
	"""
	Creates a DataStorePersistenceInitParam with default parameters, which
	does not persist the object. Any field can be overridden with a keyword
	argument
	"""

	param = datastore.DataStorePersistenceInitParam()
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID
	param.delete_last_object = True

	return override(param, fields)

def persistence_target(**fields) -> datastore.DataStorePersistenceTarget:
	"""
	Creates a DataStorePersistenceTarget with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePersistenceTarget()
	param.owner_id = 0
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID

	return override(param, fields)

def complete_post_param(**fields) -> datastore.DataStoreCompletePostParam:
	"""
	Creates a DataStoreCompletePostParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParam()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	return override(param, fields)

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

	return override(param, fields)

def change_meta_param_v1(**fields) -> datastore.DataStoreChangeMetaParamV1:
	"""
	Creates a DataStoreChangeMetaParamV1 with default parameters, which
	changes nothing. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreChangeMetaParamV1()
	param.data_id = constants.INVALID_DATAID
	param.modifies_flag = constants.ModificationFlag.MODIFICATION_FLAG_NONE
	param.name = ""
	param.permission = permission()
	param.delete_permission = permission()
	param.period = constants.DEFAULT_PERIOD
	param.meta_binary = b""
	param.tags = []
	param.update_password = constants.INVALID_PASSWORD

	return override(param, fields)

# TODO - Remove this once NintendoClients checks the revision itself, and go
#        back to datastore.DataStoreChangeMetaParam
#
# * The NintendoClients version of DataStoreChangeMetaParam always encodes
# * persistence_target, which breaks when testing on Pokemon Bank since that
# * field wasn't added until revision 1, and Bank predates class headers. This
# * would cause ChangeMetas to fail since the list of params had data the server
# * couldn't handle
class DataStoreChangeMetaParam(common.Structure):
	def __init__(self):
		super().__init__()
		self.data_id = None
		self.modifies_flag = None
		self.name = None
		self.permission = datastore.DataStorePermission()
		self.delete_permission = datastore.DataStorePermission()
		self.period = None
		self.meta_binary = None
		self.tags = None
		self.update_password = None
		self.referred_count = None
		self.data_type = None
		self.status = None
		self.compare_param = datastore.DataStoreChangeMetaCompareParam()
		self.persistence_target = datastore.DataStorePersistenceTarget()

	def __key(self):
		return (self.data_id, self.modifies_flag, self.name, self.permission, self.delete_permission, self.period, self.meta_binary, self.tags, self.update_password, self.referred_count, self.data_type, self.status, self.compare_param, self.persistence_target)

	def __hash__(self):
		return hash(common.make_hashable(self.__key()))

	def __eq__(self, other):
		if type(self) is not type(other):
			return NotImplemented
		return self.__key() == other.__key()

	def max_version(self, settings):
		# * This is a guess, Bank doesn't have this and it's NEX 3.4.7
		version = 0
		if settings["nex.version"] >= 30500:
			version = 1
		return version

	def check_required(self, settings, version):
		for field in ['data_id', 'modifies_flag', 'name', 'period', 'meta_binary', 'tags', 'update_password', 'referred_count', 'data_type', 'status']:
			if getattr(self, field) is None:
				raise ValueError("No value assigned to required field: %s" %field)

	def load(self, stream, version):
		self.data_id = stream.u64()
		self.modifies_flag = stream.u32()
		self.name = stream.string()
		self.permission = stream.extract(datastore.DataStorePermission)
		self.delete_permission = stream.extract(datastore.DataStorePermission)
		self.period = stream.u16()
		self.meta_binary = stream.qbuffer()
		self.tags = stream.list(stream.string)
		self.update_password = stream.u64()
		self.referred_count = stream.u32()
		self.data_type = stream.u16()
		self.status = stream.u8()
		self.compare_param = stream.extract(datastore.DataStoreChangeMetaCompareParam)
		if version >= 1:
			self.persistence_target = stream.extract(datastore.DataStorePersistenceTarget)

	def save(self, stream, version):
		self.check_required(stream.settings, version)
		stream.u64(self.data_id)
		stream.u32(self.modifies_flag)
		stream.string(self.name)
		stream.add(self.permission)
		stream.add(self.delete_permission)
		stream.u16(self.period)
		stream.qbuffer(self.meta_binary)
		stream.list(self.tags, stream.string)
		stream.u64(self.update_password)
		stream.u32(self.referred_count)
		stream.u16(self.data_type)
		stream.u8(self.status)
		stream.add(self.compare_param)
		if version >= 1:
			stream.add(self.persistence_target)

def change_meta_param(**fields) -> DataStoreChangeMetaParam:
	"""
	Creates a DataStoreChangeMetaParam with default parameters, which
	changes nothing. Any field can be overridden with a keyword argument
	"""

	param = DataStoreChangeMetaParam()
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

	return override(param, fields)

def complete_post_param_v1(**fields) -> datastore.DataStoreCompletePostParamV1:
	"""
	Creates a DataStoreCompletePostParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParamV1()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	return override(param, fields)

def touch_object_param(**fields) -> datastore.DataStoreTouchObjectParam:
	"""
	Creates a DataStoreTouchObjectParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreTouchObjectParam()
	param.data_id = constants.INVALID_DATAID
	param.lock_id = 0
	param.access_password = constants.INVALID_PASSWORD

	return override(param, fields)

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

	return override(param, fields)

def prepare_get_param_v1(**fields) -> datastore.DataStorePrepareGetParamV1:
	"""
	Creates a DataStorePrepareGetParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePrepareGetParamV1()
	param.data_id = constants.INVALID_DATAID
	param.lock_id = 0

	return override(param, fields)

def prepare_update_param(**fields) -> datastore.DataStorePrepareUpdateParam:
	"""
	Creates a DataStorePrepareUpdateParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePrepareUpdateParam()
	param.data_id = constants.INVALID_DATAID
	param.size = 0
	param.update_password = constants.INVALID_PASSWORD
	param.extra_data = ["WUP", "4", "EUR", "110", "GB", ""]

	return override(param, fields)

def complete_update_param(**fields) -> datastore.DataStoreCompleteUpdateParam:
	"""
	Creates a DataStoreCompleteUpdateParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompleteUpdateParam()
	param.data_id = constants.INVALID_DATAID
	param.version = 0
	param.success = True

	return override(param, fields)

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

	return override(param, fields)

def empty_meta_info() -> datastore.DataStoreMetaInfo:
	"""
	The metadata given in a batch for an object which could not be
	returned. Every field is zeroed, whatever the error was
	"""

	meta_info = datastore.DataStoreMetaInfo()
	meta_info.data_id = constants.INVALID_DATAID
	meta_info.owner_id = 0
	meta_info.size = 0
	meta_info.name = ""
	meta_info.data_type = 0
	meta_info.meta_binary = b""
	meta_info.permission = permission(permission=0)
	meta_info.delete_permission = permission(permission=0)
	meta_info.create_time = common.DateTime(0)
	meta_info.update_time = common.DateTime(0)
	meta_info.period = 0
	meta_info.status = 0
	meta_info.referred_count = 0
	meta_info.refer_data_id = 0
	meta_info.flag = 0
	meta_info.referred_time = common.DateTime(0)
	meta_info.expire_time = common.DateTime(0)
	meta_info.tags = []
	meta_info.ratings = []

	return meta_info

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

async def prepare_update_object(pid: int, **fields) -> datastore.DataStoreReqUpdateInfo:
	"""
	Starts updating an object as the given user, returning all the info.
	The update is not completed
	"""

	async with connect(pid) as client:
		return await client.prepare_update_object(prepare_update_param(**fields))

async def complete_update_object(pid: int, **fields) -> None:
	"""
	Completes an update as the given user
	"""

	async with connect(pid) as client:
		await client.complete_update_object(complete_update_param(**fields))

def rating_slot(slot: int = 0, **fields) -> datastore.DataStoreRatingInitParamWithSlot:
	"""
	A rating slot to post an object with. Any field of its settings can be
	overridden with a keyword argument
	"""

	return rating_init_param_with_slot(slot=slot, param=rating_init_param(**fields))

async def post_rated(pid: int = nex.FRIEND_A, slots: list | None = None, **fields) -> int:
	"""
	Posts an object with rating slots, slot 0 with the default settings
	unless others are given
	"""

	return await post_meta_binary(pid, rating_init_param=slots if slots is not None else [rating_slot()], **fields)

def rating_target(data_id: int, slot: int) -> datastore.DataStoreRatingTarget:
	"""
	The slot of an object to rate
	"""

	target = datastore.DataStoreRatingTarget()
	target.data_id = data_id
	target.slot = slot

	return target

def rate_object_param(rating_value: int, access_password: int) -> datastore.DataStoreRateObjectParam:
	"""
	The rating to give, and the password to give it with
	"""

	param = datastore.DataStoreRateObjectParam()
	param.rating_value = rating_value
	param.access_password = access_password

	return param

def rating_totals(info: datastore.DataStoreRatingInfo) -> tuple[int, int, int]:
	"""
	The total, count and initial value of a returned rating
	"""

	return info.total_value, info.count, info.initial_value

async def rate_object(pid: int, data_id: int, rating_value: int, slot: int = 0) -> datastore.DataStoreRatingInfo:
	"""
	Rates one slot of an object as the given user, returning what the
	slot holds afterwards
	"""

	async with connect(pid) as client:
		return await client.rate_object(rating_target(data_id, slot), rate_object_param(rating_value, constants.INVALID_PASSWORD), True)

async def rating_slot_info(data_id: int, slot: int = 0) -> tuple[int, int, int]:
	"""
	The total, count and initial value of a slot, as GetMeta gives them
	"""

	meta_info = await get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)
	[rating] = [rating for rating in meta_info.ratings if rating.slot == slot]

	return rating.info.total_value, rating.info.count, rating.info.initial_value

async def rate_object_result(pid: int, data_id: int, slot: int = 0) -> str | None:
	"""
	Rates one slot of an object, returning the name of the error, or
	None if it worked. Used to see whether a user is locked out
	"""

	async with connect(pid) as client:
		try:
			await client.rate_object(rating_target(data_id, slot), rate_object_param(1, constants.INVALID_PASSWORD), True)
		except common.RMCError as error:
			return error.name()

	return None

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

async def version_of(data_id: int) -> int:
	"""
	The current version of an object. GetMeta does not have it, so this
	uses GetSpecificMeta, which leaves out objects with a size of 0
	"""

	async with connect(nex.FRIEND_A) as client:
		param = datastore.DataStoreGetSpecificMetaParam()
		param.data_ids = [data_id]

		return (await client.get_specific_meta(param))[0].version

async def notified(data_id: int) -> list[int]:
	"""
	Every test user with an unread notification for the object
	"""

	pids = []

	for pid in [nex.FRIEND_A, nex.FRIEND_B, nex.STRANGER]:
		data_ids = []

		# * Notifications come back oldest first, a page at a time. Asking
		# * for the next page also deletes every notification before it
		async with connect(pid) as client:
			param = datastore.DataStoreGetNewArrivedNotificationsParam()
			param.last_notification_id = 0
			param.limit = 0

			while True:
				response = await client.get_new_arrived_notifications(param)
				data_ids += [notification.data_id for notification in response.result]

				if not response.has_next:
					break

				param.last_notification_id = response.result[-1].notification_id

		if data_id in data_ids:
			pids.append(pid)

	return pids

async def notification_ids(pid: int, data_id: int) -> list[int]:
	"""
	The IDs of the users unread notifications for the object, to tell a
	new notification apart from an old one
	"""

	async with connect(pid) as client:
		param = datastore.DataStoreGetNewArrivedNotificationsParam()
		param.last_notification_id = 0
		param.limit = 99

		notifications = (await client.get_new_arrived_notifications(param)).result

	return [notification.notification_id for notification in notifications if notification.data_id == data_id]

async def post_persisted(post, pid: int, slot: int, **fields) -> int:
	"""
	Posts an object into one of the users persistence slots,
	using the given post method, and returns its data ID
	"""

	init_param = persistence_init_param(persistence_id=slot)

	return await post(pid, persistence_init_param=init_param, **fields)

async def post_rejected(post, pid: int, **fields) -> int:
	"""
	Posts an object and marks it as rejected, as the given user
	"""

	data_id = await post(pid, **fields)

	await change_meta(pid, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=constants.DataStatus.DATA_STATUS_REJECTED)

	return data_id

def rating_slots(count: int, first: int = 0, **fields) -> list:
	"""
	A list of rating slot params, starting at the given slot
	"""

	return [rating_init_param_with_slot(slot=first + i, param=rating_init_param(**fields)) for i in range(count)]

async def post_with_before_metadata(post, pid: int, **fields) -> int:
	"""
	Posts an object with predetermined metadata, to detect changes
	"""

	fields.setdefault("name", BEFORE_NAME)
	fields.setdefault("meta_binary", BEFORE_META_BINARY)
	fields.setdefault("tags", BEFORE_TAGS)
	fields.setdefault("period", BEFORE_PERIOD)

	return await post(pid, **fields)

# * A value for each field which the posted object never holds
MISMATCHED_VALUES = {
	"name": "other",
	"permission": permission(permission=constants.Permission.PERMISSION_FRIEND),
	"delete_permission": permission(permission=constants.Permission.PERMISSION_PUBLIC),
	"period": BEFORE_PERIOD + 1,
	"meta_binary": b"OTHER",
	"tags": ["other"],
	"referred_count": 1,
	"data_type": constants.INVALID_DATA_TYPE - 1, # * Far past any unique_data_type
	"status": constants.DataStatus.DATA_STATUS_PENDING
}

# * Every way an object can be created, in general. Used when general object
# * creation in the database is needed
POST_METHODS = [
	pytest.param(post_meta_binary, id="post-meta-binary"),
	pytest.param(post_object, id="post-object"),
	pytest.param(post_object_v1, id="post-object-v1")
]

# * Every way an object can be created with a persistence slot set. Used for
# * when a test needs to test slots
PERSISTENCE_POST_METHODS = [
	pytest.param(post_meta_binary, id="post-meta-binary"),
	pytest.param(post_object, id="post-object")
]

# * Used to ensure that tests involving objects which use the file server
# * like PrepareGetObject, are ran against objects posted through both
# * ways of creating objects on the file server
FILE_SERVER_POST_METHODS = [
	pytest.param(post_object, id="post-object"),
	pytest.param(post_object_v1, id="post-object-v1")
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
