import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def change_metas(pid: int, data_ids: list[int], entries: list[dict], transactional: bool) -> list:
	"""
	Changes the metadata of several objects as the given user, returning
	the result of each one. Each entry is the fields of one param
	"""

	async with helpers.connect(pid) as client:
		params = [helpers.change_meta_param_v1(**entry) for entry in entries]

		return await client.change_metas_v1(data_ids, params, transactional)

async def change_metas_error(pid: int, data_ids: list[int], entries: list[dict], transactional: bool) -> str:
	"""
	Changes the metadata of several objects and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		params = [helpers.change_meta_param_v1(**entry) for entry in entries]

		try:
			await client.change_metas_v1(data_ids, params, transactional)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::ChangeMetasV1 did not throw an error")

# * Every ChangeMetaV1 test is run again here to ensure both methods change
# * objects the same way at a base level

async def change(pid: int, **fields) -> None:
	"""
	Changes an objects metadata as the given user, as the only param of
	a request, and expects it to work
	"""

	assert helpers.result_names(await change_metas(pid, [], [fields], False)) == [None]

async def change_error(pid: int, **fields) -> str:
	"""
	Changes an objects metadata as the only param of a request, and
	expects it to fail, returning the name of the error. That is the
	result of the one param, or the error thrown for the whole request
	when it cannot be read at all
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			results = await client.change_metas_v1([], [helpers.change_meta_param_v1(**fields)], False)
		except common.RMCError as error:
			return error.name()

	if results[0].is_success():
		pytest.fail("DataStoreProtocol::ChangeMetasV1 did not fail")

	return results[0].name()

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="zero"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	A data ID with no object behind it is missing
	"""

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after") == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after") == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects have no metadata to change
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after") == "DataStore::NotFound"

# * modifies_flag

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_only_the_fields_in_the_flag_are_changed(unique_data_type, post):
	"""
	Every field is sent every time, but only the ones named in the flag
	are applied
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	access = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])
	update = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after", permission=access, delete_permission=update, period=30, meta_binary=b"AFTER", tags=["after"])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert info.name == "after"
	assert info.permission == helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	assert info.delete_permission == helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	assert info.period == helpers.BEFORE_PERIOD
	assert info.meta_binary == helpers.BEFORE_META_BINARY
	assert info.tags == helpers.BEFORE_TAGS

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_no_flag_changes_nothing(unique_data_type, post):
	"""
	A request which asks for nothing is not an error, it just does
	nothing
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NONE, name="after")

	assert await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_several_flags_at_once(unique_data_type, post):
	"""
	The flag is a bit field, so more than one field can be changed in
	one request
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	flag = constants.ModificationFlag.MODIFICATION_FLAG_NAME | constants.ModificationFlag.MODIFICATION_FLAG_TAGS

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=flag, name="after", tags=["after"])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert info.name == "after"
	assert info.tags == ["after"]
	assert info.meta_binary == helpers.BEFORE_META_BINARY

@pytest.mark.parametrize("flag", helpers.FLAGS_WITH_NO_FIELD)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_flag_with_no_field_behind_it(unique_data_type, flag, post):
	"""
	V1 has no field for these, so asking for them is taken but cannot
	do anything
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=flag)

	assert await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_data_type_flag_is_refused(unique_data_type, post):
	"""
	Unlike the other fields V1 is missing, asking to change the data
	type is refused rather than ignored?
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_DATA_TYPE) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_flag_which_is_not_defined(unique_data_type, post):
	"""
	Bits which mean nothing are ignored rather than refused
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=0x400, name="after") # * Unused flag

	assert await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

# * modifies_flag, the update time and the expiry
# *
# * The update time and the expiry move together, and which of them
# * moves depends on the two flags which touch them

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_the_period_moves_the_expiry(unique_data_type, post):
	"""
	The expiry is worked out from the update time, which this does not
	touch, so it moves without the update time moving
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=30)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.period == 30
	assert after.update_time.value() == before.update_time.value()
	assert after.expire_time.value() < before.expire_time.value()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_the_updated_time_moves_the_expiry_too(unique_data_type, post):
	"""
	Marking the object as updated restarts its period
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.period == before.period
	assert after.update_time.value() > before.update_time.value()
	assert after.expire_time.value() > before.expire_time.value()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_both_the_period_and_the_updated_time(unique_data_type, post):
	"""
	With both, the period starts again from now using the new number
	of days
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	flag = constants.ModificationFlag.MODIFICATION_FLAG_PERIOD | constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=flag, period=30)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.period == 30
	assert after.update_time.value() > before.update_time.value()
	assert after.expire_time.value() < before.expire_time.value()

async def test_changing_the_updated_time_does_not_send_an_update_notification(unique_data_type):
	"""
	Only updating the objects data with CompleteUpdateObject sends an
	update notification. Marking the object as updated would replace the
	unread post notification if it sent one
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	flag = constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST | constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE
	data_id = await helpers.post_with_before_metadata(helpers.post_object, nex.FRIEND_A, data_type=unique_data_type, permission=permission, flag=flag)
	before = await helpers.notification_ids(nex.STRANGER, data_id)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME)

	assert len(before) == 1
	assert await helpers.notification_ids(nex.STRANGER, data_id) == before

# * name

@pytest.mark.parametrize("name", [
	pytest.param("n" * constants.MAX_NAME_LENGTH, id="at-the-limit"),
	pytest.param("", id="empty")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_which_is_allowed(unique_data_type, name, post):
	"""
	Anything up to MAX_NAME_LENGTH is allowed, including nothing
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name=name)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == name

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_over_the_limit(unique_data_type, post):
	"""
	A name longer than MAX_NAME_LENGTH is a failure
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	name = "n" * (constants.MAX_NAME_LENGTH + 1)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name=name) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("text", helpers.TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_in_any_script(unique_data_type, text, post):
	"""
	Every sample which fits in 16 bit characters is stored and handed
	back unchanged
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name=text)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == text

@pytest.mark.parametrize("text", helpers.ASTRAL_TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_above_the_basic_plane(unique_data_type, text, post):
	"""
	Emoji and anything else needing a surrogate pair cannot be decoded,
	which fails before DataStore ever sees the request
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name=text) == "PythonCore::ConversionError"

@pytest.mark.parametrize("text", helpers.CONTROL_TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_with_control_characters(unique_data_type, text, post):
	"""
	A name keeps control characters as they are, unlike a tag
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name=text)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == text

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_name_with_a_null_in_it(unique_data_type, post):
	"""
	Strings are null terminated on the wire, so a null in the middle
	ends the name early
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="a\x00b")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "a"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_name_limit_counts_characters_not_bytes(unique_data_type, post):
	"""
	Length is measured in code units, not bytes
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="\u6f22" * constants.MAX_NAME_LENGTH)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="\u6f22" * (constants.MAX_NAME_LENGTH + 1)) == "DataStore::InvalidArgument"

# * permission and delete_permission

@pytest.mark.parametrize("permission, recipients", helpers.PERMISSIONS)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_the_access_permission(unique_data_type, permission, recipients, post):
	"""
	Every permission can be set, along with its recipients
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	access = helpers.permission(permission=permission, recipients=recipients)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_ACCESS_PERMISSION, permission=access)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert info.permission == access

@pytest.mark.parametrize("permission, recipients", helpers.PERMISSIONS)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_the_update_permission(unique_data_type, permission, recipients, post):
	"""
	Every permission can be set, along with its recipients
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	update = helpers.permission(permission=permission, recipients=recipients)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATE_PERMISSION, delete_permission=update)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert info.delete_permission == update

@pytest.mark.parametrize("flag, field", [
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_ACCESS_PERMISSION, "permission", id="access"),
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_UPDATE_PERMISSION, "delete_permission", id="update")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_unknown_permission(unique_data_type, flag, field, post):
	"""
	Only the permissions which exist can be used
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	bad = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND + 1)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=flag, **{field: bad}) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("flag, field", [
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_ACCESS_PERMISSION, "permission", id="access"),
	pytest.param(constants.ModificationFlag.MODIFICATION_FLAG_UPDATE_PERMISSION, "delete_permission", id="update")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_recipients_on_a_permission_which_does_not_use_them(unique_data_type, flag, field, post):
	"""
	Recipients only mean something to the specified permissions, so
	anywhere else they are refused
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	bad = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC, recipients=[nex.STRANGER])

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=flag, **{field: bad}) == "DataStore::InvalidArgument"

async def test_changing_the_access_permission_keeps_the_notifications(unique_data_type):
	"""
	A notification is not taken back from a recipient who loses access,
	and nobody who gains access is notified
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_with_before_metadata(helpers.post_meta_binary, nex.FRIEND_A, data_type=unique_data_type, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST)
	access = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])

	assert await helpers.notified(data_id) == [nex.STRANGER]

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_ACCESS_PERMISSION, permission=access)

	assert await helpers.notified(data_id) == [nex.STRANGER]

# * period

@pytest.mark.parametrize("period", [
	pytest.param(constants.MAX_PERIOD, id="at-the-limit"),
	pytest.param(1, id="one-day")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_period_which_is_allowed(unique_data_type, period, post):
	"""
	Anything up to MAX_PERIOD is allowed
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=period)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).period == period

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_period_over_the_limit(unique_data_type, post):
	"""
	Anything past MAX_PERIOD is a failure
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=constants.MAX_PERIOD + 1) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_period_of_zero_expires_the_object(unique_data_type, post):
	"""
	The same as posting with no period
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * meta_binary

@pytest.mark.parametrize("meta_binary", [
	pytest.param(b"m" * constants.MAX_METABIN_SIZE, id="at-the-limit"),
	pytest.param(b"", id="empty")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_meta_binary_which_is_allowed(unique_data_type, meta_binary, post):
	"""
	Anything up to MAX_METABIN_SIZE is allowed, including nothing
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_METABINARY, meta_binary=meta_binary)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_METABINARY)

	assert info.meta_binary == meta_binary

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_meta_binary_over_the_limit(unique_data_type, post):
	"""
	A meta binary larger than MAX_METABIN_SIZE is a failure
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	meta_binary = b"m" * (constants.MAX_METABIN_SIZE + 1)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_METABINARY, meta_binary=meta_binary) == "DataStore::InvalidArgument"

# * tags

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_tags_are_sorted_when_they_are_stored(unique_data_type, post):
	"""
	Tags are sorted by byte value on a change, the same as on a post
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=["9nine", "banana", "1one", "Apple"])
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == ["1one", "9nine", "Apple", "banana"]

@pytest.mark.parametrize("tags", [
	pytest.param(["tag%i" % i for i in range(constants.NUM_TAG_SLOT)], id="at-the-slot-limit"),
	pytest.param(["t" * constants.MAX_TAG_LENGTH], id="a-tag-at-the-length-limit"),
	pytest.param([], id="none-at-all")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_tags_which_are_allowed(unique_data_type, tags, post):
	"""
	Every tag slot can be filled, and an empty list clears them
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=tags)
	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.tags == sorted(tags)

@pytest.mark.parametrize("tags", [
	pytest.param(["tag%i" % i for i in range(constants.NUM_TAG_SLOT + 1)], id="over-the-slot-limit"),
	pytest.param(["t" * (constants.MAX_TAG_LENGTH + 1)], id="a-tag-over-the-length-limit"),
	pytest.param(["same", "same"], id="duplicates")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_tags_which_are_not_allowed(unique_data_type, tags, post):
	"""
	The same limits apply as when the object was posted
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=tags) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("text", helpers.TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_tag_in_any_script(unique_data_type, text, post):
	"""
	Tags take the same range of characters names do
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=[text])

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)).tags == [text]

@pytest.mark.parametrize("text", helpers.ASTRAL_TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_tag_above_the_basic_plane(unique_data_type, text, post):
	"""
	A tag which cannot be decoded fails the same way a name does
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=[text]) == "PythonCore::ConversionError"

@pytest.mark.parametrize("text", helpers.CONTROL_TEXT_SAMPLES)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_tag_with_control_characters(unique_data_type, text, post):
	"""
	Control characters are refused in tags
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=[text]) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_tag_limit_counts_characters_not_bytes(unique_data_type, post):
	"""
	The tag size limit is counted the same way the name size limit is
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=["\u6f22" * constants.MAX_TAG_LENGTH])

	assert await change_error(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=["\u6f22" * (constants.MAX_TAG_LENGTH + 1)]) == "DataStore::InvalidArgument"

# * update_password
# *
# * Changing metadata is gated by the objects update permission

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_only_the_owner_can_change_a_private_object(unique_data_type, post):
	"""
	The update permission decides who may change the metadata
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after") == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_anybody_can_change_an_object_which_allows_it(unique_data_type, post):
	"""
	An update permission of public lets anybody change the metadata
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type, delete_permission=update)

	await change(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_right_update_password_gets_in(unique_data_type, post):
	"""
	The password stands in for the update permission
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	await change(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after", update_password=password_info.update_password)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_wrong_update_password(unique_data_type, post):
	"""
	Sending a password which is not the right one says so
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await change_error(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after", update_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_access_password_is_not_the_update_password(unique_data_type, post):
	"""
	Passwords only work in the place they belong to
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await change_error(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after", update_password=password_info.access_password) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_wrong_update_password_from_the_owner(unique_data_type, post):
	"""
	The password is only looked at when it is needed
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after", update_password=1234)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "after"

# * The object status

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_pending_object_can_be_changed_by_its_owner(unique_data_type, post):
	"""
	An object under review can still be changed by its owner
	"""

	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	await change(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Everyone but the owner is told it is under review, whatever their
	update permission is
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await change_error(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after") == "DataStore::UnderReviewing"

# * Changing the metadata does not count as referring to the object

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changing_the_metadata_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move, even when somebody other
	than the owner makes the change
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_with_before_metadata(post, nex.FRIEND_A, data_type=unique_data_type, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await change(nex.STRANGER, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after")

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.expire_time.value() == before.expire_time.value()

# * Tests specific to the batch version

# * data_ids and params

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_one_param_for_several_data_ids(unique_data_type, post):
	"""
	The one param is applied to every object in data_ids
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [first, second], entries, False)) == [None, None]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).name == "after"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=second)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_param_names_its_own_object(unique_data_type, post):
	"""
	With no data_ids, each param changes the object its data_id names
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"data_id": first,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		},
		{
			"data_id": second,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [], entries, False)) == [None, None]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).name == "after"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=second)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_data_ids_are_used_over_the_param_data_id(unique_data_type, post):
	"""
	When data_ids is given, the data_id in the param is ignored
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"data_id": helpers.UNKNOWN_DATA_ID,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [data_id], entries, False)) == [None]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_data_id_of_zero_does_not_fall_back_on_the_param(unique_data_type, post):
	"""
	A data ID of 0 in data_ids is missing, even when the param names a
	real object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	entries = [
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [constants.INVALID_DATAID], entries, False)) == ["DataStore::NotFound"]
	assert await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("data_ids, entries", [
	pytest.param([helpers.UNKNOWN_DATA_ID, helpers.UNKNOWN_DATA_ID], [{}, {}], id="as-many-params-as-data-ids"),
	pytest.param([helpers.UNKNOWN_DATA_ID], [{}, {}], id="more-params-than-data-ids"),
	pytest.param([helpers.UNKNOWN_DATA_ID], [], id="data-ids-and-no-params"),
	pytest.param([], [], id="nothing-at-all")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_a_request_of_neither_shape(data_ids, entries, transactional):
	"""
	Several params alongside data_ids, data_ids with no param, or no
	params at all, is a bad request
	"""

	assert await change_metas_error(nex.FRIEND_A, data_ids, entries, transactional) == "DataStore::InvalidArgument"

# * data_ids and params, at the capacity
# *
# * BATCH_PROCESSING_CAPACITY limits the number of objects in one request

@pytest.mark.parametrize("transactional", [True, False])
async def test_exactly_the_capacity_of_data_ids(transactional):
	"""
	The capacity itself is allowed
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * constants.BATCH_PROCESSING_CAPACITY

	if transactional:
		assert await change_metas_error(nex.FRIEND_A, data_ids, [{}], transactional) == "DataStore::NotFound"
	else:
		assert helpers.result_names(await change_metas(nex.FRIEND_A, data_ids, [{}], transactional)) == ["DataStore::NotFound"] * len(data_ids)

@pytest.mark.parametrize("transactional", [True, False])
async def test_too_many_data_ids(transactional):
	"""
	One object over the capacity is a failure, always
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * (constants.BATCH_PROCESSING_CAPACITY + 1)

	assert await change_metas_error(nex.FRIEND_A, data_ids, [{}], transactional) == "DataStore::OverCapacity"

@pytest.mark.parametrize("transactional", [True, False])
async def test_exactly_the_capacity_of_params(transactional):
	"""
	The capacity itself is allowed
	"""

	entries = [{"data_id": helpers.UNKNOWN_DATA_ID}] * constants.BATCH_PROCESSING_CAPACITY

	if transactional:
		assert await change_metas_error(nex.FRIEND_A, [], entries, transactional) == "DataStore::NotFound"
	else:
		assert helpers.result_names(await change_metas(nex.FRIEND_A, [], entries, transactional)) == ["DataStore::NotFound"] * len(entries)

@pytest.mark.parametrize("transactional", [True, False])
async def test_too_many_params(transactional):
	"""
	One object over the capacity is a failure, always
	"""

	entries = [{"data_id": helpers.UNKNOWN_DATA_ID}] * (constants.BATCH_PROCESSING_CAPACITY + 1)

	assert await change_metas_error(nex.FRIEND_A, [], entries, transactional) == "DataStore::OverCapacity"

# * params
# *
# * Every param is a change of its own, with its own flag and password

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_param_uses_its_own_flag(unique_data_type, post):
	"""
	Two params changing different fields of one object both apply
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		},
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_TAGS,
			"tags": ["after"]
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [], entries, False)) == [None, None]

	info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert info.name == "after"
	assert info.tags == ["after"]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_param_uses_its_own_password(unique_data_type, post):
	"""
	The password is read off each param, not just the first one
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	first_password = (await helpers.get_password_info(nex.FRIEND_A, first)).update_password
	second_password = (await helpers.get_password_info(nex.FRIEND_A, second)).update_password

	entries = [
		{
			"data_id": first,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after",
			"update_password": first_password
		},
		{
			"data_id": second,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after",
			"update_password": second_password
		}
	]

	assert helpers.result_names(await change_metas(nex.STRANGER, [], entries, False)) == [None, None]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_wrong_password_only_fails_its_own_entry(unique_data_type, post):
	"""
	A bad password fails that entry alone, and says the password was wrong
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)
	password = (await helpers.get_password_info(nex.FRIEND_A, first)).update_password

	entries = [
		{
			"data_id": first,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after",
			"update_password": password
		},
		{
			"data_id": second,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after",
			"update_password": 1234
		}
	]

	assert helpers.result_names(await change_metas(nex.STRANGER, [], entries, False)) == [None, "DataStore::InvalidPassword"]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_one_password_for_several_data_ids(unique_data_type, post):
	"""
	With one param for several data IDs, its password only gets in to
	the object it belongs to
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)
	password = (await helpers.get_password_info(nex.FRIEND_A, first)).update_password

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after",
			"update_password": password
		}
	]

	assert helpers.result_names(await change_metas(nex.STRANGER, [first, second], entries, False)) == [None, "DataStore::InvalidPassword"]

# * transactional, set to true
# *
# * A transactional request is all or nothing. It throws the first error
# * it runs into, leaves every object alone, and has no per-object
# * results to give back

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_changes_every_object(unique_data_type, post):
	"""
	Every object is changed, and there are no results because nothing
	can have gone wrong
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert await change_metas(nex.FRIEND_A, [first, second], entries, True) == []
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).name == "after"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=second)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("position", [
	pytest.param(0, id="the-failure-comes-first"),
	pytest.param(1, id="the-failure-comes-last")
])
async def test_transactional_leaves_everything_alone_when_one_fails(unique_data_type, post, position):
	"""
	One object which cannot be changed fails the whole request, rolling
	back any changes, regardless of where the failure happened
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)
	data_ids = [helpers.UNKNOWN_DATA_ID, data_id] if position == 0 else [data_id, helpers.UNKNOWN_DATA_ID]

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert await change_metas_error(nex.FRIEND_A, data_ids, entries, True) == "DataStore::NotFound"
	assert await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_rolls_back_a_permission_failure(unique_data_type, post):
	"""
	An object the caller may not change fails the request, and their
	own object in the same list is left alone
	"""

	mine = await post(nex.STRANGER, data_type=unique_data_type)
	theirs = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.STRANGER, data_id=mine, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert await change_metas_error(nex.STRANGER, [mine, theirs], entries, True) == "DataStore::PermissionDenied"
	assert await helpers.get_meta(nex.STRANGER, data_id=mine, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_rolls_back_an_invalid_change(unique_data_type, post):
	"""
	A param which would make an invalid change fails the request, and
	the valid change before it is undone
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=first, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	entries = [
		{
			"data_id": first,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		},
		{
			"data_id": second,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "n" * (constants.MAX_NAME_LENGTH + 1)
		}
	]

	assert await change_metas_error(nex.FRIEND_A, [], entries, True) == "DataStore::InvalidArgument"
	assert await helpers.get_meta(nex.FRIEND_A, data_id=first, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("unknown_first, error", [
	pytest.param(True, "DataStore::NotFound", id="the-unknown-object-first"),
	pytest.param(False, "DataStore::PermissionDenied", id="the-forbidden-object-first")
])
async def test_transactional_throws_the_first_failure(unique_data_type, post, unknown_first, error):
	"""
	With more than one failure in the list, the error thrown is the one
	which comes first
	"""

	theirs = await post(nex.FRIEND_A, data_type=unique_data_type)
	data_ids = [helpers.UNKNOWN_DATA_ID, theirs] if unknown_first else [theirs, helpers.UNKNOWN_DATA_ID]

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	assert await change_metas_error(nex.STRANGER, data_ids, entries, True) == error

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_the_same_object_twice(unique_data_type, post):
	"""
	The params are applied in order, so the last change to an object
	is the one it keeps
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "first"
		},
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "second"
		}
	]

	assert await change_metas(nex.FRIEND_A, [], entries, True) == []
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "second"

# * transactional, set to false
# *
# * Without the transactional flag each object is dealt with on its own,
# * the request always succeeds, and the results list says what happened
# * to each one

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_changes_what_it_can(unique_data_type, post):
	"""
	The objects which can be changed are changed
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	results = await change_metas(nex.FRIEND_A, [first, helpers.UNKNOWN_DATA_ID, second], entries, False)

	assert helpers.result_names(results) == [None, "DataStore::NotFound", None]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).name == "after"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=second)).name == "after"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_permission_failure_only_fails_its_own_entry(unique_data_type, post):
	"""
	An object the caller may not change fails on its own, leaving the
	rest of the list to go through
	"""

	mine = await post(nex.STRANGER, data_type=unique_data_type)
	theirs = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=theirs, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		}
	]

	results = await change_metas(nex.STRANGER, [mine, theirs], entries, False)

	assert helpers.result_names(results) == [None, "DataStore::PermissionDenied"]
	assert (await helpers.get_meta(nex.STRANGER, data_id=mine)).name == "after"
	assert await helpers.get_meta(nex.FRIEND_A, data_id=theirs, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_invalid_change_only_fails_its_own_entry(unique_data_type, post):
	"""
	A param which would make an invalid change fails on its own
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=second, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	entries = [
		{
			"data_id": first,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "after"
		},
		{
			"data_id": second,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "n" * (constants.MAX_NAME_LENGTH + 1)
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [], entries, False)) == [None, "DataStore::InvalidArgument"]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).name == "after"
	assert await helpers.get_meta(nex.FRIEND_A, data_id=second, result_option=constants.ResultFlag.RESULT_FLAG_ALL) == before

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_invalid_param_fails_every_data_id(unique_data_type, post):
	"""
	With one param for several data IDs, an invalid change fails for
	every one of them
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "n" * (constants.MAX_NAME_LENGTH + 1)
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [first, second], entries, False)) == ["DataStore::InvalidArgument", "DataStore::InvalidArgument"]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_same_object_twice(unique_data_type, post):
	"""
	The params are applied in order, so the last change to an object
	is the one it keeps
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "first"
		},
		{
			"data_id": data_id,
			"modifies_flag": constants.ModificationFlag.MODIFICATION_FLAG_NAME,
			"name": "second"
		}
	]

	assert helpers.result_names(await change_metas(nex.FRIEND_A, [], entries, False)) == [None, None]
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "second"

# * Objects which cannot be changed, in a transactional request

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="zero"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_transactional_a_data_id_which_names_nothing(data_id):
	"""
	A missing object is thrown for the whole request
	"""

	assert await change_metas_error(nex.FRIEND_A, [data_id], [{}], True) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_a_pending_object_is_under_review(unique_data_type, post):
	"""
	An object under review is thrown for the whole request
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await change_metas_error(nex.STRANGER, [data_id], [{}], True) == "DataStore::UnderReviewing"
