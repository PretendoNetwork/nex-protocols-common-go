import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Every stock flag. Unlike PreparePostObject, an object which is not on
# * the file server takes DATA_FLAG_NOT_USE_FILESERVER, but refuses
# * DATA_FLAG_USE_NOTIFICATION_ON_UPDATE as it can never be updated
DATA_FLAGS = [
	pytest.param(constants.DataFlag.DATA_FLAG_NEED_REVIEW, True, id="need-review"),
	pytest.param(constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED, True, id="period-from-last-referred"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_READ_LOCK, True, id="use-read-lock"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST, True, id="use-notification-on-post"),
	pytest.param(constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE, False, id="use-notification-on-update"),
	pytest.param(constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER, True, id="not-use-fileserver"),
	pytest.param(constants.DataFlag.DATA_FLAG_NEED_COMPLETION, True, id="need-completion")
]

async def post_error(pid: int, **fields) -> str:
	"""
	Posts an object which is not on the file server and expects an
	error to be thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.post_meta_binary(helpers.post_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PostMetaBinary did not throw an error")

# * What the new object starts out as

async def test_the_object_is_there_straight_away(unique_data_type):
	"""
	There is nothing to upload, so unlike PreparePostObject the object
	exists as soon as it is posted, for everybody it lets in
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id
	assert (await helpers.get_meta(nex.STRANGER, data_id=data_id)).data_id == data_id

async def test_every_object_gets_its_own_data_id(unique_data_type):
	"""
	Two objects posted the same way do not collide
	"""

	first = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	second = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert first != second

async def test_the_object_belongs_to_the_caller(unique_data_type):
	"""
	Whoever posts the object owns it
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).owner_id == nex.FRIEND_A

async def test_the_object_is_marked_as_not_on_the_file_server(unique_data_type):
	"""
	DATA_FLAG_NOT_USE_FILESERVER is set by the server, without the
	caller asking for it
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).flag == constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

async def test_a_new_object_has_never_been_referred_to(unique_data_type):
	"""
	Posting an object is not a reference, so it starts with no
	references, and a referred time of when it was created
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.referred_count == 0
	assert meta_info.referred_time.value() == meta_info.create_time.value()

# * size
# *
# * The meta binary is the object, so there is no file whose size could
# * be given. A size of 0 is the only one accepted

async def test_the_size_is_zero(unique_data_type):
	"""
	The size stays at 0, rather than becoming the size of the meta binary
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, size=0, meta_binary=b"meta")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == 0

@pytest.mark.parametrize("size, meta_binary", [
	pytest.param(4, b"meta", id="the-size-of-the-meta-binary"),
	pytest.param(1, b"", id="with-no-meta-binary")
])
async def test_a_size_which_is_not_zero(unique_data_type, size, meta_binary):
	"""
	Any other size is refused, even the size of the meta binary
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, size=size, meta_binary=meta_binary) == "DataStore::InvalidArgument"

# * name

async def test_a_name_at_the_limit(unique_data_type):
	"""
	A name of exactly MAX_NAME_LENGTH is allowed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="n" * constants.MAX_NAME_LENGTH)

	assert data_id != constants.INVALID_DATAID

async def test_a_name_over_the_limit(unique_data_type):
	"""
	A name longer than MAX_NAME_LENGTH is a failure
	"""

	name = "n" * (constants.MAX_NAME_LENGTH + 1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, name=name) == "DataStore::InvalidArgument"

async def test_an_empty_name(unique_data_type):
	"""
	Objects do not have to be named
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == ""

async def test_the_name_is_kept_on_the_object(unique_data_type):
	"""
	The name is stored as it was sent
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="a name")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "a name"

# * name, as a string
# *
# * Object strings are UTF-8 on the wire but 16 bit characters on the server

@pytest.mark.parametrize("text", helpers.TEXT_SAMPLES)
async def test_a_name_in_any_script(unique_data_type, text):
	"""
	Every sample which fits in 16 bit characters is stored and handed
	back unchanged
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name=text)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == text

@pytest.mark.parametrize("text", helpers.ASTRAL_TEXT_SAMPLES)
async def test_a_name_above_the_basic_plane(unique_data_type, text):
	"""
	Emoji and anything else needing a surrogate pair cannot be decoded,
	which fails before DataStore ever sees the request
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, name=text) == "PythonCore::ConversionError"

@pytest.mark.parametrize("text", helpers.CONTROL_TEXT_SAMPLES)
async def test_a_name_with_control_characters(unique_data_type, text):
	"""
	A name takes control characters and keeps them as they are, unlike
	a tag, which refuses them?
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name=text)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == text

async def test_a_name_with_a_null_in_it(unique_data_type):
	"""
	Strings are null terminated on the wire, so a null in the middle
	ends the name early rather than being refused
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="a\x00b")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "a"

@pytest.mark.parametrize("text, allowed", [
	pytest.param("漢" * constants.MAX_NAME_LENGTH, True, id="at-the-limit"),
	pytest.param("漢" * (constants.MAX_NAME_LENGTH + 1), False, id="one-over-the-limit")
])
async def test_the_name_limit_counts_characters_not_bytes(unique_data_type, text, allowed):
	"""
	Length is measured in code units, not bytes
	"""

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, name=text) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name=text)

	assert data_id != constants.INVALID_DATAID

# * data_type

@pytest.mark.parametrize("data_type", [
	pytest.param(0, id="the-lowest"),
	pytest.param(constants.INVALID_DATA_TYPE - 1, id="the-highest-real-one")
])
async def test_a_data_type_which_is_allowed(data_type):
	"""
	Everything below the invalid one is a usable data type
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=data_type)

	assert data_id != constants.INVALID_DATAID

async def test_the_invalid_data_type():
	"""
	An invalid data type is invalid
	"""

	assert await post_error(nex.FRIEND_A, data_type=constants.INVALID_DATA_TYPE) == "DataStore::InvalidArgument"

async def test_the_data_type_is_kept_on_the_object(unique_data_type):
	"""
	The data type is stored as it was sent
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_type == unique_data_type

# * meta_binary

async def test_a_meta_binary_at_the_limit(unique_data_type):
	"""
	A meta binary size MAX_METABIN_SIZE is allowed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"m" * constants.MAX_METABIN_SIZE)

	assert data_id != constants.INVALID_DATAID

async def test_a_meta_binary_over_the_limit(unique_data_type):
	"""
	A meta binary larger than MAX_METABIN_SIZE is a failure
	"""

	meta_binary = b"m" * (constants.MAX_METABIN_SIZE + 1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, meta_binary=meta_binary) == "DataStore::InvalidArgument"

async def test_an_empty_meta_binary(unique_data_type):
	"""
	The object does not have to have any contents
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"")
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_METABINARY)

	assert info.meta_binary == b""

async def test_the_meta_binary_is_kept_on_the_object(unique_data_type):
	"""
	The meta binary comes back byte for byte
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"\x00\xFF\x10binary")
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_METABINARY)

	assert info.meta_binary == b"\x00\xFF\x10binary"

# * permission

@pytest.mark.parametrize("permission, recipients", helpers.PERMISSIONS)
async def test_an_access_permission_is_kept_on_the_object(unique_data_type, permission, recipients):
	"""
	Every access permission is allowed, and both it and its recipients
	are stored as they were sent
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=access)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert info.permission.permission == permission
	assert info.permission.recipients == recipients

async def test_an_unknown_access_permission(unique_data_type):
	"""
	Only the permissions which exist can be used
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND + 1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, permission=permission) == "DataStore::InvalidArgument"

async def test_access_recipients_on_a_permission_which_does_not_use_them(unique_data_type):
	"""
	Recipients only mean something to the specified permissions, so
	anywhere else they are refused
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC, recipients=[nex.STRANGER])

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, permission=permission) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("count", [
	pytest.param(constants.DATASTOREPERMISSION_RECIPIENTIDS_MAX, id="at-the-limit"),
	pytest.param(constants.DATASTOREPERMISSION_RECIPIENTIDS_MAX + 1, id="one-over-the-limit")
])
async def test_how_many_access_recipients_are_allowed(unique_data_type, count):
	"""
	DATASTOREPERMISSION_RECIPIENTIDS_MAX seems to not be enforced on Nintendo Network.
	Possibly only added on the Switch/NEX 4?
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=list(range(1, count + 1)))
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert data_id != constants.INVALID_DATAID

# * delete_permission

@pytest.mark.parametrize("permission, recipients", helpers.PERMISSIONS)
async def test_a_delete_permission_is_kept_on_the_object(unique_data_type, permission, recipients):
	"""
	The delete permission is stored the same way the access one is,
	and the two do not interfere
	"""

	delete_permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, delete_permission=delete_permission)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert info.delete_permission.permission == permission
	assert info.delete_permission.recipients == recipients
	assert info.permission.permission == constants.Permission.PERMISSION_PUBLIC

async def test_an_unknown_delete_permission(unique_data_type):
	"""
	Only the permissions which exist can be used
	"""

	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND + 1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, delete_permission=delete_permission) == "DataStore::InvalidArgument"

async def test_delete_recipients_on_a_permission_which_does_not_use_them(unique_data_type):
	"""
	The delete permission refuses stray recipients the same way
	"""

	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE, recipients=[nex.STRANGER])

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, delete_permission=delete_permission) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("count", [
	pytest.param(constants.DATASTOREPERMISSION_RECIPIENTIDS_MAX, id="at-the-limit"),
	pytest.param(constants.DATASTOREPERMISSION_RECIPIENTIDS_MAX + 1, id="one-over-the-limit")
])
async def test_how_many_delete_recipients_are_allowed(unique_data_type, count):
	"""
	The delete permission recipients are not counted either
	"""

	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=list(range(1, count + 1)))
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, delete_permission=delete_permission)

	assert data_id != constants.INVALID_DATAID

# * flag

@pytest.mark.parametrize("flag, allowed", DATA_FLAGS)
async def test_every_data_flag(unique_data_type, flag, allowed):
	"""
	Every flag which is defined can be set, besides the one which needs
	an upload to happen
	"""

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, flag=flag) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=flag)

	assert data_id != constants.INVALID_DATAID

@pytest.mark.parametrize("flag", [pytest.param(param.values[0], id=param.id) for param in DATA_FLAGS if param.values[1] and param.values[0] != constants.DataFlag.DATA_FLAG_NEED_COMPLETION])
async def test_a_data_flag_is_kept_on_the_object(unique_data_type, flag):
	"""
	The flags which are allowed come back as they were sent, along with
	DATA_FLAG_NOT_USE_FILESERVER. An object which needs completing
	cannot be seen yet, so that flag is tested on its own
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=flag)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).flag == flag | constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

async def test_a_flag_which_is_not_defined(unique_data_type):
	"""
	Unknown flags are rejected
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, flag=0x80) == "DataStore::InvalidArgument" # * The first bit no flag uses

async def test_two_flags_at_once(unique_data_type):
	"""
	Flags are a bit field, so more than one can be set
	"""

	flag = constants.DataFlag.DATA_FLAG_NEED_REVIEW | constants.DataFlag.DATA_FLAG_USE_READ_LOCK
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=flag)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).flag == flag | constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

async def test_the_need_review_flag_sets_the_status(unique_data_type):
	"""
	Asking for a review is what puts an object into the pending state
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).status == constants.DataStatus.DATA_STATUS_PENDING

async def test_the_need_completion_flag_hides_the_object_until_it_is_completed(unique_data_type):
	"""
	Without the flag there is nothing to complete, but with it the object
	cannot be seen until CompletePostObject is called on it, the same as
	an object on the file server
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_COMPLETION)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

	async with helpers.connect(nex.FRIEND_A) as client:
		await client.complete_post_object(helpers.complete_post_param(data_id=data_id))

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id

@pytest.mark.parametrize("permission, recipients, notified", helpers.NOTIFIED_PERMISSIONS)
async def test_the_notification_flag_notifies_the_recipients_on_post(unique_data_type, permission, recipients, notified):
	"""
	There is no upload to finish, so the recipients are notified as soon
	as the object is posted
	"""

	access_permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=access_permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST)

	assert await helpers.notified(data_id) == notified

async def test_no_notification_without_the_notification_flag(unique_data_type):
	"""
	Recipients are only notified when the object asks for it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await helpers.notified(data_id) == []

async def test_the_notification_flag_does_not_notify_delete_recipients(unique_data_type):
	"""
	Only the access permission decides who is notified
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission, delete_permission=delete_permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST)

	assert await helpers.notified(data_id) == []

async def test_the_notification_flag_notifies_for_an_object_which_needs_review(unique_data_type):
	"""
	The recipients are notified even though the object is still pending
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	flag = constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST | constants.DataFlag.DATA_FLAG_NEED_REVIEW
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission, flag=flag)

	assert await helpers.notified(data_id) == [nex.STRANGER]

async def test_the_notification_flag_waits_for_the_need_completion_flag(unique_data_type):
	"""
	An object which needs completion is not posted until it is completed,
	so the recipients are not notified until then either
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	flag = constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST | constants.DataFlag.DATA_FLAG_NEED_COMPLETION
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission, flag=flag)

	assert await helpers.notified(data_id) == []

	async with helpers.connect(nex.FRIEND_A) as client:
		await client.complete_post_object(helpers.complete_post_param(data_id=data_id))

	assert await helpers.notified(data_id) == [nex.STRANGER]

# * period

@pytest.mark.parametrize("period", [
	pytest.param(constants.MAX_PERIOD, id="at-the-limit"),
	pytest.param(0, id="zero")
])
async def test_a_period_which_is_allowed(unique_data_type, period):
	"""
	Anything up to the maximum is allowed, including no period at all
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, period=period)

	assert data_id != constants.INVALID_DATAID

async def test_a_period_which_is_not_allowed(unique_data_type):
	"""
	Anything past the maximum is invalid
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, period=constants.MAX_PERIOD + 1) == "DataStore::InvalidArgument"

async def test_a_period_of_zero_expires_the_object_at_once(unique_data_type):
	"""
	The period is counted in days from when the object was made, so a period
	of 0 expires the object immediately
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, period=0)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_the_period_is_kept_on_the_object(unique_data_type):
	"""
	The period is stored as it was sent, and the expiry is worked out
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, period=constants.MAX_PERIOD)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert info.period == constants.MAX_PERIOD
	assert info.expire_time.value() > info.create_time.value() # TODO - Better check for this

# * refer_data_id

@pytest.mark.parametrize("refer_data_id", [
	pytest.param(constants.INVALID_DATAID, id="zero"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="an-object-which-is-not-there")
])
async def test_a_refer_data_id_is_not_checked(unique_data_type, refer_data_id):
	"""
	Accepts any value without checks it seems
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, refer_data_id=refer_data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).refer_data_id == refer_data_id

async def test_a_refer_data_id_which_is_a_real_object(unique_data_type):
	"""
	Referring to an object which does exist works the same way
	"""

	referred = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, refer_data_id=referred)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).refer_data_id == referred

async def test_referring_to_an_object_is_not_a_reference(unique_data_type):
	"""
	The object referred to keeps its reference count and referred time
	"""

	referred = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=referred)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, refer_data_id=referred)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=referred)

	assert after.referred_count == before.referred_count
	assert after.referred_time == before.referred_time
	assert after.expire_time == before.expire_time

# * tags

async def test_tags_at_the_slot_limit(unique_data_type):
	"""
	An object can fill every tag slot it has
	"""

	tags = ["tag%i" % i for i in range(constants.NUM_TAG_SLOT)]
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=tags)

	assert data_id != constants.INVALID_DATAID

async def test_tags_over_the_slot_limit(unique_data_type):
	"""
	One tag more than there are slots is refused
	"""

	tags = ["tag%i" % i for i in range(constants.NUM_TAG_SLOT + 1)]

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=tags) == "DataStore::InvalidArgument"

async def test_a_tag_at_the_length_limit(unique_data_type):
	"""
	A tag of exactly the maximum length is allowed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=["t" * constants.MAX_TAG_LENGTH])

	assert data_id != constants.INVALID_DATAID

async def test_a_tag_over_the_length_limit(unique_data_type):
	"""
	One character more than the maximum is refused
	"""

	tags = ["t" * (constants.MAX_TAG_LENGTH + 1)]

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=tags) == "DataStore::InvalidArgument"

async def test_duplicate_tags(unique_data_type):
	"""
	The same tag cannot be used twice on one object
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=["same", "same"]) == "DataStore::InvalidArgument"

async def test_an_empty_tag(unique_data_type):
	"""
	A tag with nothing in it is allowed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=[""])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == [""]

async def test_the_tags_are_sorted_when_they_are_stored(unique_data_type):
	"""
	Tags are sorted on the server by byte value
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=["9nine", "banana", "1one", "Apple"])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == ["1one", "9nine", "Apple", "banana"]

async def test_the_tags_are_not_trimmed(unique_data_type):
	"""
	Whitespace around a tag is part of the tag, and is left alone
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=["  padded  "])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == ["  padded  "]

@pytest.mark.parametrize("text", helpers.TEXT_SAMPLES)
async def test_a_tag_in_any_script(unique_data_type, text):
	"""
	Tags take the same range of characters names do
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=[text])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == [text]

@pytest.mark.parametrize("text", helpers.ASTRAL_TEXT_SAMPLES)
async def test_a_tag_above_the_basic_plane(unique_data_type, text):
	"""
	A tag which cannot be decoded fails the same way a name does
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=[text]) == "PythonCore::ConversionError"

@pytest.mark.parametrize("text", helpers.CONTROL_TEXT_SAMPLES)
async def test_a_tag_with_control_characters(unique_data_type, text):
	"""
	Control characters are refused in tags
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=[text]) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("text, allowed", [
	pytest.param("漢" * constants.MAX_TAG_LENGTH, True, id="at-the-limit"),
	pytest.param("漢" * (constants.MAX_TAG_LENGTH + 1), False, id="one-over-the-limit")
])
async def test_the_tag_limit_counts_characters_not_bytes(unique_data_type, text, allowed):
	"""
	The tag size limit is counted the same way the name size limit is
	"""

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, tags=[text]) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, tags=[text])

	assert data_id != constants.INVALID_DATAID

# * rating_init_param

async def test_a_rating_slot_is_set_up(unique_data_type):
	"""
	A rating slot should get created when posted
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(1))
	ratings = (await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)).ratings

	assert [rating.slot for rating in ratings] == [0]
	assert ratings[0].info.total_value == 0
	assert ratings[0].info.count == 0

async def test_a_rating_slot_can_start_at_a_value(unique_data_type):
	"""
	The initial value is the value the slot starts at, and it does not
	count as somebody having rated
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(1, initial_value=50))
	ratings = (await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)).ratings

	assert ratings[0].info.total_value == 50
	assert ratings[0].info.count == 0

async def test_rating_slots_at_the_limit(unique_data_type):
	"""
	An object can set up every rating slot it has
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(constants.NUM_RATING_SLOT))

	assert data_id != constants.INVALID_DATAID

async def test_rating_slots_over_the_limit(unique_data_type):
	"""
	One rating slot more than there are is refused
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(constants.NUM_RATING_SLOT + 1)) == "DataStore::InvalidArgument"

async def test_a_rating_slot_which_does_not_exist(unique_data_type):
	"""
	Only slots 0 to NUM_RATING_SLOT - 1 can be set up
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(1, first=constants.NUM_RATING_SLOT)) == "DataStore::InvalidArgument"

async def test_two_rating_params_for_the_same_slot(unique_data_type):
	"""
	A slot can only be set up once, and asking twice is refused rather
	than the second one winning
	"""

	params = helpers.rating_slots(1) + helpers.rating_slots(1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::OperationNotAllowed"

@pytest.mark.parametrize("lock_type", helpers.RATING_LOCK_TYPES)
async def test_every_rating_lock_type(unique_data_type, lock_type):
	"""
	Every lock type which is defined can be set up
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(1, lock_type=lock_type))

	# * Lock types aren't exposed anywhere to the client after being made
	# * so we have to just trust this worked
	assert data_id != constants.INVALID_DATAID

async def test_a_rating_lock_type_which_does_not_exist(unique_data_type):
	"""
	Only the lock types which exist can be used
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT + 1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("fields", [
	pytest.param({"range_min": 1, "range_max": 10}, id="a-range"),
	pytest.param({"range_min": 10, "range_max": 1}, id="a-range-the-wrong-way-round"),
	pytest.param({"range_min": 1, "range_max": 10, "initial_value": 100}, id="an-initial-value-outside-the-range"),
	pytest.param({"flag": 0xFF}, id="every-rating-flag"),
	pytest.param({"internal_flag": 0xFF}, id="every-internal-flag"),
	pytest.param({"period_hour": 25}, id="an-hour-which-is-not-an-hour"),
	pytest.param({"period_duration": -17}, id="a-negative-period-duration")
])
async def test_the_rest_of_a_rating_param_is_not_checked(unique_data_type, fields):
	"""
	Without the range flags, or a lock type which uses the duration and
	hour, the rest is taken however it comes
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=helpers.rating_slots(1, **fields))

	assert data_id != constants.INVALID_DATAID

@pytest.mark.parametrize("internal_flag, allowed", [
	pytest.param(constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MIN | constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MAX, False, id="both-ends"),
	pytest.param(constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MIN, True, id="only-the-minimum"),
	pytest.param(constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MAX, True, id="only-the-maximum")
])
async def test_a_range_the_wrong_way_round(unique_data_type, internal_flag, allowed):
	"""
	A range whose minimum is above its maximum is only refused when both
	ends of it are used
	"""

	params = helpers.rating_slots(1, internal_flag=internal_flag, range_min=10, range_max=1)

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params)

	assert data_id != constants.INVALID_DATAID

async def test_an_interval_lock_with_a_negative_duration(unique_data_type):
	"""
	An interval lock cannot last for less than no time at all
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_INTERVAL, period_duration=-1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"

# * Positive values up to 365 are also taken, presumably as a number of days
@pytest.mark.parametrize("period_duration, allowed", [
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, True, id="monday"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_SUN, True, id="sunday"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_SUN - 1, False, id="past-sunday"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_DAY1 - 1, False, id="past-the-first-of-the-month"),
	pytest.param(0, True, id="zero"),
	pytest.param(365, True, id="365"),
	pytest.param(366, False, id="366")
])
async def test_the_day_a_period_lock_lifts(unique_data_type, period_duration, allowed):
	"""
	A period lock only takes days it knows
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=period_duration)

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params)

	assert data_id != constants.INVALID_DATAID

# * RATING_LOCK_PERIOD_DAY1 is a real value, and works as described on the
# * wiki, but Pokemon Bank (a NEX 3.4 server) crashes on it. It is assumed to
# * have been added in NEX 3.5, like several other things Pokemon Bank lacks

@pytest.mark.skipif(nex.SERVER_NEX_VERSION >= 30500, reason="RATING_LOCK_PERIOD_DAY1 was added in NEX 3.5")
async def test_a_period_lock_on_the_first_of_the_month_before_nex_3_5(unique_data_type):
	"""
	The server recognizes RATING_LOCK_PERIOD_DAY1, but crashes on it
	rather than refusing it
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=constants.RatingLockPeriod.RATING_LOCK_PERIOD_DAY1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "PythonCore::Exception"

# * Disclaimer: these tests were not ran against a live server, as at the time
# * of writing the only active NEX server on Nintendo Network is Pokemon Bank,
# * which is a NEX 3.4 server. The tests are written to assume the current
# * implementation is correct, and will need to be updated at some point when
# * more testing is done on NSO

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="RATING_LOCK_PERIOD_DAY1 was added in NEX 3.5")
@pytest.mark.parametrize("period_hour", [
	pytest.param(0, id="midnight"),
	pytest.param(-5, id="a-negative-hour")
])
async def test_a_period_lock_on_the_first_of_the_month(unique_data_type, period_hour):
	"""
	RATING_LOCK_PERIOD_DAY1 can be used, including with a negative hour,
	which goes back into the last day of the month before
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=constants.RatingLockPeriod.RATING_LOCK_PERIOD_DAY1, period_hour=period_hour)

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params)

	assert data_id != constants.INVALID_DATAID

@pytest.mark.parametrize("period_hour, allowed", [
	pytest.param(23, True, id="the-last-hour"),
	pytest.param(-23, True, id="the-last-hour-going-back"),
	pytest.param(24, False, id="past-the-last-hour"),
	pytest.param(-24, False, id="past-the-last-hour-going-back")
])
async def test_the_hour_a_period_lock_lifts(unique_data_type, period_hour, allowed):
	"""
	A period lock only takes hours between -23 and 23
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, period_hour=period_hour)

	if not allowed:
		assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"
		return

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params)

	assert data_id != constants.INVALID_DATAID

async def test_a_negative_hour_with_a_number_of_days(unique_data_type):
	"""
	A negative hour counts back from midnight of a day of the week, so it
	is refused when period_duration is a number of days instead
	"""

	params = helpers.rating_slots(1, lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=1, period_hour=-1)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("fields", [
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_INTERVAL, "period_duration": 5, "period_hour": 25}, id="an-interval-lock-with-an-hour-which-is-not-an-hour"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERMANENT, "period_duration": -1, "period_hour": 25}, id="a-permanent-lock-with-a-negative-duration-and-an-hour-which-is-not-an-hour")
])
async def test_lock_settings_are_only_checked_for_the_lock_which_uses_them(unique_data_type, fields):
	"""
	The duration and hour are only checked by the lock types which use
	them
	"""

	params = helpers.rating_slots(1, **fields)

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=params)

	assert data_id != constants.INVALID_DATAID

# * persistence_init_param

async def test_a_persistence_slot_puts_the_object_in_it(unique_data_type):
	"""
	An object can go straight into a slot, where it never expires
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=0)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, 0)).data_id == data_id
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time.value() == constants.PERMANENT_DATE_TIME

async def test_no_persistence_slot(unique_data_type):
	"""
	The invalid slot id means no slot at all, rather than a bad one
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=constants.INVALID_PERSISTENCE_SLOT_ID)
	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param)

	assert data_id != constants.INVALID_DATAID

async def test_a_persistence_slot_which_does_not_exist(unique_data_type):
	"""
	Only slots 0 to NUM_PERSISTENCE_SLOT - 1 can be used
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=constants.NUM_PERSISTENCE_SLOT)

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param) == "DataStore::InvalidArgument"

async def test_replacing_an_object_deletes_the_old_one(unique_data_type):
	"""
	With delete_last_object, the object which was in the slot is deleted
	"""

	old = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=helpers.persistence_init_param(persistence_id=0))
	new = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=helpers.persistence_init_param(persistence_id=0, delete_last_object=True))

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, 0)).data_id == new
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=old) == "DataStore::NotFound"

async def test_replacing_an_object_can_keep_the_old_one(unique_data_type):
	"""
	Without delete_last_object, the object which was in the slot is
	kept, and expires again like any other
	"""

	old = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=helpers.persistence_init_param(persistence_id=0))
	new = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=helpers.persistence_init_param(persistence_id=0, delete_last_object=False))

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, 0)).data_id == new
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=old)).expire_time.value() != constants.PERMANENT_DATE_TIME

@pytest.mark.parametrize("delete_last_object, notified", [
	pytest.param(True, [], id="deleting-the-old-object"),
	pytest.param(False, [nex.STRANGER], id="keeping-the-old-object")
])
async def test_replacing_an_object_and_its_notifications(unique_data_type, delete_last_object, notified):
	"""
	The notifications for the object which was in the slot are deleted
	along with it, and kept when it is kept
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	old = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST, persistence_init_param=helpers.persistence_init_param(persistence_id=0))

	assert await helpers.notified(old) == [nex.STRANGER]

	await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=helpers.persistence_init_param(persistence_id=0, delete_last_object=delete_last_object))

	assert await helpers.notified(old) == notified

# * extra_data
# *
# * Disclaimer: these tests were not ran against a live server, as at the time
# * of writing the only active NEX server on Nintendo Network is Pokemon Bank,
# * which is a NEX 3.4 server, and this field was added in NEX 3.5. The tests
# * are written to assume the current implementation is correct, and will need
# * to be updated at some point when more testing is done on NSO

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
async def test_no_extra_data_at_all(unique_data_type):
	"""
	A client older than NEX 3.5 does not send the field, which is not
	the same as sending something wrong
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, extra_data=[])

	assert data_id != constants.INVALID_DATAID

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "0", "JPN", "1", "JP", ""], id="a-japanese-3ds"),
	pytest.param(["CTR", "2", "EUR", "110", "GB", ""], id="a-european-3ds"),
	pytest.param(["CTR", "6", "TWN", "128", "TW", ""], id="a-taiwanese-3ds"),
	pytest.param(["WUP", "2", "USA", "49", "US", ""], id="an-american-wii-u"),
	pytest.param(["WUP", "4", "EUR", "110", "GB", ""], id="a-european-wii-u"),
	pytest.param(["WUP", "64", "TWN", "128", "TW", ""], id="a-taiwanese-wii-u"),
	pytest.param(["WUP", "0", "Invalid", "0", "", ""], id="a-wii-u-which-does-not-know-its-region")
])
async def test_extra_data_a_console_would_send(unique_data_type, extra_data):
	"""
	Both platforms and each of their regions are accepted
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data)

	assert data_id != constants.INVALID_DATAID

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "2", "EUR", "110", "GB"], id="one-too-few"),
	pytest.param(["CTR", "2", "EUR", "110", "GB", "", ""], id="one-too-many")
])
async def test_extra_data_of_the_wrong_length(unique_data_type, extra_data):
	"""
	There are always exactly six elements when the field is sent at all
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
async def test_an_extra_data_string_which_is_too_long(unique_data_type):
	"""
	The console allocates eight bytes for each of these, one of which
	is the trailing null, so seven characters is the most which fits
	"""

	extra_data = ["CTR", "2", "EUR", "110", "GB", "x" * 8]

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
async def test_an_extra_data_string_at_the_length_limit(unique_data_type):
	"""
	Seven characters is allowed, which is what "Invalid" needs
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, extra_data=["CTR", "2", "EUR", "110", "GB", "x" * 7])

	assert data_id != constants.INVALID_DATAID

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["XYZ", "2", "EUR", "110", "GB", ""], id="a-platform-which-does-not-exist"),
	pytest.param(["", "2", "EUR", "110", "GB", ""], id="no-platform-at-all"),
	pytest.param(["ctr", "2", "EUR", "110", "GB", ""], id="the-right-platform-in-lower-case")
])
async def test_extra_data_with_an_unknown_platform(unique_data_type, extra_data):
	"""
	Only the 3DS and the Wii U are known platforms
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "two", "EUR", "110", "GB", ""], id="a-region-id-which-is-not-a-number"),
	pytest.param(["CTR", "", "EUR", "110", "GB", ""], id="no-region-id-at-all")
])
async def test_extra_data_with_a_region_id_which_is_not_a_number(unique_data_type, extra_data):
	"""
	The region ID is a number written out as a string
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "2", "ZZZ", "110", "GB", ""], id="a-region-which-does-not-exist"),
	pytest.param(["CTR", "0", "Invalid", "0", "", ""], id="a-region-only-the-wii-u-has")
])
async def test_extra_data_with_an_unknown_region(unique_data_type, extra_data):
	"""
	The region name has to be one the platform actually has
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "4", "EUR", "110", "GB", ""], id="a-3ds-using-the-wii-u-number-for-europe"),
	pytest.param(["WUP", "2", "EUR", "110", "GB", ""], id="a-wii-u-using-the-3ds-number-for-europe"),
	pytest.param(["CTR", "1", "EUR", "110", "GB", ""], id="a-number-for-a-different-region")
])
async def test_extra_data_where_the_region_id_and_name_disagree(unique_data_type, extra_data):
	"""
	The two platforms number their regions differently, so the ID has
	to be the one which goes with the name on that platform
	"""

	assert await post_error(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data) == "DataStore::InvalidArgument"

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "2", "EUR", "9999", "GB", ""], id="a-country-id-which-does-not-exist"),
	pytest.param(["CTR", "2", "EUR", "110", "ZZ", ""], id="a-country-code-which-does-not-match"),
	pytest.param(["CTR", "2", "EUR", "", "", ""], id="no-country-at-all"),
	pytest.param(["CTR", "2", "EUR", "110", "GB", "spare"], id="something-in-the-spare-element")
])
async def test_the_parts_of_extra_data_which_are_not_checked(unique_data_type, extra_data):
	"""
	Only the platform and its region are looked at, so the country and
	the spare element are taken however they come
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, extra_data=extra_data)

	assert data_id != constants.INVALID_DATAID
