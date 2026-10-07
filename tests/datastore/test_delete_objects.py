import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT = 3

async def delete(pid: int, entries: list[dict], transactional: bool) -> list:
	"""
	Deletes several objects as the given user, returning the result of
	each one. Each entry is the fields of one delete param
	"""

	async with helpers.connect(pid) as client:
		params = [helpers.delete_param(**entry) for entry in entries]

		return await client.delete_objects(params, transactional)

async def delete_error(pid: int, entries: list[dict], transactional: bool) -> str:
	"""
	Deletes several objects and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		params = [helpers.delete_param(**entry) for entry in entries]

		try:
			await client.delete_objects(params, transactional)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::DeleteObjects did not throw an error")

# * Requests which never get as far as looking at an object

@pytest.mark.parametrize("transactional", [True, False])
async def test_no_data_ids(transactional):
	"""
	Deleting nothing at all is not an error
	"""

	assert await delete(nex.FRIEND_A, [], transactional) == []

@pytest.mark.parametrize("transactional", [True, False])
async def test_exactly_the_capacity(transactional):
	"""
	The capacity itself is allowed
	"""

	entries = [{"data_id": helpers.UNKNOWN_DATA_ID}] * constants.BATCH_PROCESSING_CAPACITY

	if transactional:
		assert await delete_error(nex.FRIEND_A, entries, transactional) == "DataStore::NotFound"
	else:
		assert helpers.result_names(await delete(nex.FRIEND_A, entries, transactional)) == ["DataStore::NotFound"] * len(entries)

@pytest.mark.parametrize("transactional", [True, False])
async def test_too_many_data_ids(transactional):
	"""
	One object over the capacity is a failure, always
	"""

	entries = [{"data_id": helpers.UNKNOWN_DATA_ID}] * (constants.BATCH_PROCESSING_CAPACITY + 1)

	assert await delete_error(nex.FRIEND_A, entries, transactional) == "DataStore::OverCapacity"

# * A transactional request is all or nothing. It throws the first error
# * it runs into, leaves every object alone, and has no per-object
# * results to give back

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_deletes_every_object(unique_data_type, post):
	"""
	Every object in the list is deleted, and there are no results
	because nothing can have gone wrong
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await delete(nex.FRIEND_A, [{"data_id": first}, {"data_id": second}], True) == []
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=first) == "DataStore::NotFound"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=second) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("position", [
	pytest.param(0, id="the-failure-comes-first"),
	pytest.param(1, id="the-failure-comes-last")
])
async def test_transactional_leaves_everything_alone_when_one_fails(unique_data_type, post, position):
	"""
	One object which cannot be deleted fails the whole request, rolling
	back any changes, regardless of where the failure happened
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	entries = [{"data_id": helpers.UNKNOWN_DATA_ID}, {"data_id": data_id}] if position == 0 else [{"data_id": data_id}, {"data_id": helpers.UNKNOWN_DATA_ID}]

	assert await delete_error(nex.FRIEND_A, entries, True) == "DataStore::NotFound"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_rolls_back_a_permission_failure(unique_data_type, post):
	"""
	An object the caller may not delete fails the request, and their
	own object in the same list is left alone
	"""

	mine = await post(nex.FRIEND_A, data_type=unique_data_type)
	theirs = await post(nex.STRANGER, data_type=unique_data_type)

	assert await delete_error(nex.FRIEND_A, [{"data_id": mine}, {"data_id": theirs}], True) == "DataStore::PermissionDenied"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=mine)).data_id == mine

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_transactional_the_same_object_twice(unique_data_type, post):
	"""
	The second time an object appears it has already been deleted by
	the first, which fails the request and reverts the first deletion
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await delete_error(nex.FRIEND_A, [{"data_id": data_id}, {"data_id": data_id}], True) == "DataStore::NotFound"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id

# * Without the transactional flag each object is dealt with on its own,
# * the request always succeeds, and the results list say what happened
# * to each one in turn

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deletes_what_it_can(unique_data_type, post):
	"""
	The objects which can be deleted are deleted
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	results = await delete(nex.FRIEND_A, [{"data_id": first}, {"data_id": helpers.UNKNOWN_DATA_ID}, {"data_id": second}], False)

	assert helpers.result_names(results) == [None, "DataStore::NotFound", None]
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=first) == "DataStore::NotFound"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=second) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_permission_failure_only_fails_its_own_entry(unique_data_type, post):
	"""
	An object the caller may not delete fails on its own, leaving the
	rest of the list to go through
	"""

	mine = await post(nex.FRIEND_A, data_type=unique_data_type)
	theirs = await post(nex.STRANGER, data_type=unique_data_type)

	results = await delete(nex.FRIEND_A, [{"data_id": mine}, {"data_id": theirs}], False)

	assert helpers.result_names(results) == [None, "DataStore::PermissionDenied"]
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=mine) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_same_object_twice(unique_data_type, post):
	"""
	The second time an object appears it has already been deleted by
	the first, so only that entry fails
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	results = await delete(nex.FRIEND_A, [{"data_id": data_id}, {"data_id": data_id}], False)

	assert helpers.result_names(results) == [None, "DataStore::NotFound"]
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * Objects which are in an unusual state

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deleting_an_object_under_review(unique_data_type, post):
	"""
	Objects which are under review are hidden from everybody, but
	their owner can still delete them
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert helpers.result_names(await delete(nex.FRIEND_A, [{"data_id": data_id}], False)) == [None]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deleting_a_read_locked_object(unique_data_type, post):
	"""
	Read locks only gate downloads of an object, so one which is
	locked by somebody else can still be deleted
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.STRANGER, data_id=data_id, lock_id=1234)

	assert helpers.result_names(await delete(nex.FRIEND_A, [{"data_id": data_id}], False)) == [None]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_deleting_a_persisted_object_empties_the_slot(unique_data_type, post):
	"""
	Deleting an object which is being persisted leaves its
	persistence slot empty
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param)

	assert helpers.result_names(await delete(nex.FRIEND_A, [{"data_id": data_id}], False)) == [None]

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT) == "DataStore::NotFound"

# * Data IDs which never name an object

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(999999, id="the-system-range"),
	pytest.param(helpers.UINT64_MAX, id="uint64-max")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_data_ids_which_do_not_exist(data_id, transactional):
	"""
	Objects which are not there are reported as missing, the same way
	DeleteObject reports them
	"""

	if transactional:
		assert await delete_error(nex.FRIEND_A, [{"data_id": data_id}], transactional) == "DataStore::NotFound"
	else:
		assert helpers.result_names(await delete(nex.FRIEND_A, [{"data_id": data_id}], transactional)) == ["DataStore::NotFound"]

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet, so
	it cannot be deleted
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert helpers.result_names(await delete(nex.FRIEND_A, [{"data_id": data_id}], False)) == ["DataStore::NotFound"]

# * Each entry carries its own update password, which stands in for the
# * delete permission the same way it does for DeleteObject

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_entry_uses_its_own_password(unique_data_type, post):
	"""
	The password is read off each entry, not just the first one
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	first_password = (await helpers.get_password_info(nex.FRIEND_A, first)).update_password
	second_password = (await helpers.get_password_info(nex.FRIEND_A, second)).update_password

	entries = [
		{"data_id": first, "update_password": first_password},
		{"data_id": second, "update_password": second_password}
	]

	assert helpers.result_names(await delete(nex.STRANGER, entries, False)) == [None, None]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_wrong_password_only_fails_its_own_entry(unique_data_type, post):
	"""
	A bad password fails that entry alone, and says the password was wrong
	"""

	deletable = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	public = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=deletable)
	private = await post(nex.FRIEND_A, data_type=unique_data_type)

	entries = [{"data_id": public}, {"data_id": private, "update_password": 1234}]
	results = await delete(nex.STRANGER, entries, False)

	assert helpers.result_names(results) == [None, "DataStore::InvalidPassword"]
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=public) == "DataStore::NotFound"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=private)).data_id == private
