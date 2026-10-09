import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def reset(pid: int, data_ids: list[int], transactional: bool) -> list:
	"""
	Resets every slot of several objects as the given user, returning the
	result of each one
	"""

	async with helpers.connect(pid) as client:
		return await client.reset_ratings(data_ids, transactional)

async def reset_error(pid: int, data_ids: list[int], transactional: bool) -> str:
	"""
	Resets every slot of several objects and expects an error to be
	thrown for the whole request, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.reset_ratings(data_ids, transactional)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::ResetRatings did not throw an error")

async def post_rated(pid: int = nex.FRIEND_A, **fields) -> int:
	"""
	Posts an object with slot 0 starting at 10, and rates it, so a reset
	can be seen
	"""

	data_id = await helpers.post_rated(pid, slots=[helpers.rating_slot(initial_value=10)], **fields)
	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	return data_id

# * What a reset does

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_slot_of_every_object_is_reset(unique_data_type, post):
	"""
	Every users ratings are removed from every slot of every object, so
	each total goes back to its initial value and nothing is counted
	"""

	slots = [helpers.rating_slot(0, initial_value=10), helpers.rating_slot(3, initial_value=30)]
	first = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=slots)
	second = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(initial_value=20)])

	await helpers.rate_object(nex.FRIEND_A, first, 5)
	await helpers.rate_object(nex.STRANGER, first, 3, slot=3)
	await helpers.rate_object(nex.FRIEND_A, second, 7)
	await reset(nex.FRIEND_A, [first, second], False)

	assert await helpers.rating_slot_info(first, 0) == (10, 0, 10)
	assert await helpers.rating_slot_info(first, 3) == (30, 0, 30)
	assert await helpers.rating_slot_info(second, 0) == (20, 0, 20)

async def test_the_slots_can_be_rated_again_after_a_reset(unique_data_type):
	"""
	Ratings start counting again from the initial value
	"""

	data_id = await post_rated(data_type=unique_data_type)

	await reset(nex.FRIEND_A, [data_id], False)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 2)) == (12, 1, 10)

@pytest.mark.parametrize("slots", [
	pytest.param([helpers.rating_slot(initial_value=4)], id="a-slot-which-was-never-rated"),
	pytest.param([], id="no-slots-at-all")
])
async def test_an_object_with_nothing_to_reset(unique_data_type, slots):
	"""
	There is nothing to remove
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=slots)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == [None]

# * data_ids

async def test_no_data_ids():
	"""
	Asking for nothing does nothing
	"""

	assert await reset(nex.FRIEND_A, [], False) == []

@pytest.mark.parametrize("transactional", [True, False])
async def test_exactly_the_capacity(unique_data_type, transactional):
	"""
	BATCH_PROCESSING_CAPACITY data IDs are allowed
	"""

	data_id = await post_rated(data_type=unique_data_type)

	await reset(nex.FRIEND_A, [data_id] * constants.BATCH_PROCESSING_CAPACITY, transactional)

	assert await helpers.rating_slot_info(data_id) == (10, 0, 10)

@pytest.mark.parametrize("transactional", [True, False])
async def test_too_many_data_ids(unique_data_type, transactional):
	"""
	More than BATCH_PROCESSING_CAPACITY data IDs fails the whole request,
	and nothing is reset
	"""

	data_id = await post_rated(data_type=unique_data_type)

	assert await reset_error(nex.FRIEND_A, [data_id] * (constants.BATCH_PROCESSING_CAPACITY + 1), transactional) == "DataStore::OverCapacity"
	assert await helpers.rating_slot_info(data_id) == (15, 1, 10)

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to reset
	"""

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert helpers.result_names(await reset(nex.FRIEND_A, [999999], False)) == ["DataStore::NotFound"]

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects have no ratings to reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot()])

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

# * transactional, set to true
# *
# * A transactional request is all or nothing. It throws the first error
# * it runs into, leaves every object alone, and has no per-object
# * results to give back

async def test_transactional_resets_every_object(unique_data_type):
	"""
	Every object is reset, and there are no results because nothing can
	have gone wrong
	"""

	first = await post_rated(data_type=unique_data_type)
	second = await post_rated(data_type=unique_data_type)

	assert await reset(nex.FRIEND_A, [first, second], True) == []
	assert await helpers.rating_slot_info(first) == (10, 0, 10)
	assert await helpers.rating_slot_info(second) == (10, 0, 10)

@pytest.mark.parametrize("position", [
	pytest.param(0, id="the-failure-comes-first"),
	pytest.param(1, id="the-failure-comes-last")
])
async def test_transactional_leaves_everything_alone_when_one_fails(unique_data_type, position):
	"""
	One object which cannot be reset fails the whole request, rolling
	back any changes, regardless of where the failure happened
	"""

	data_id = await post_rated(data_type=unique_data_type)
	data_ids = [data_id, helpers.UNKNOWN_DATA_ID]

	if position == 0:
		data_ids.reverse()

	assert await reset_error(nex.FRIEND_A, data_ids, True) == "DataStore::NotFound"
	assert await helpers.rating_slot_info(data_id) == (15, 1, 10)

async def test_transactional_rolls_back_a_permission_failure(unique_data_type):
	"""
	An object the caller may not reset fails the request, and their own
	object in the same list is left alone
	"""

	mine = await post_rated(data_type=unique_data_type)
	theirs = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type)

	assert await reset_error(nex.FRIEND_A, [mine, theirs], True) == "DataStore::PermissionDenied"
	assert await helpers.rating_slot_info(mine) == (15, 1, 10)

@pytest.mark.parametrize("order, error", [
	pytest.param("permission-first", "DataStore::PermissionDenied", id="permission-first"),
	pytest.param("missing-first", "DataStore::NotFound", id="missing-first")
])
async def test_transactional_throws_the_first_error(unique_data_type, order, error):
	"""
	When more than one object fails, the error is the one for whichever
	comes first in the list
	"""

	theirs = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type)
	data_ids = [theirs, helpers.UNKNOWN_DATA_ID]

	if order == "missing-first":
		data_ids.reverse()

	assert await reset_error(nex.FRIEND_A, data_ids, True) == error

async def test_transactional_the_same_object_twice(unique_data_type):
	"""
	Resetting an object which was just reset is not an error, so the same
	object can appear more than once
	"""

	data_id = await post_rated(data_type=unique_data_type)

	assert await reset(nex.FRIEND_A, [data_id, data_id], True) == []
	assert await helpers.rating_slot_info(data_id) == (10, 0, 10)

# * transactional, set to false
# *
# * Without the transactional flag each object is dealt with on its own,
# * the request always succeeds, and the results list says what happened
# * to each one in turn

async def test_resets_what_it_can(unique_data_type):
	"""
	The objects which can be reset are reset
	"""

	first = await post_rated(data_type=unique_data_type)
	second = await post_rated(data_type=unique_data_type)
	results = await reset(nex.FRIEND_A, [first, helpers.UNKNOWN_DATA_ID, second], False)

	assert helpers.result_names(results) == [None, "DataStore::NotFound", None]
	assert await helpers.rating_slot_info(first) == (10, 0, 10)
	assert await helpers.rating_slot_info(second) == (10, 0, 10)

async def test_a_permission_failure_only_fails_its_own_entry(unique_data_type):
	"""
	An object the caller may not reset is left alone, and their own
	object is still reset
	"""

	mine = await post_rated(data_type=unique_data_type)
	theirs = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type)
	results = await reset(nex.FRIEND_A, [mine, theirs], False)

	assert helpers.result_names(results) == [None, "DataStore::PermissionDenied"]
	assert await helpers.rating_slot_info(mine) == (10, 0, 10)

async def test_the_same_object_twice(unique_data_type):
	"""
	Each data ID gets its own result, even when it is the same object
	"""

	data_id = await post_rated(data_type=unique_data_type)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id, data_id], False)) == [None, None]

# * The objects update permission
# *
# * Resetting is gated by the objects update permission, which is separate
# * from the access permission used to read it. There is no update password
# * to get around it with

@pytest.mark.parametrize("permission, recipients, pid, allowed", helpers.UPDATE_PERMISSIONS)
async def test_the_update_permission(unique_data_type, permission, recipients, pid, allowed):
	"""
	Only users the update permission allows can reset the object, and a
	refused reset leaves the ratings alone
	"""

	update = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post_rated(data_type=unique_data_type, delete_permission=update)
	results = await reset(pid, [data_id], False)

	if not allowed:
		assert helpers.result_names(results) == ["DataStore::PermissionDenied"]
		assert await helpers.rating_slot_info(data_id) == (15, 1, 10)
		return

	assert helpers.result_names(results) == [None]
	assert await helpers.rating_slot_info(data_id) == (10, 0, 10)

async def test_the_access_permission_is_not_used(unique_data_type):
	"""
	Someone who may update an object can reset it even if they cannot
	read it
	"""

	access = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await post_rated(data_type=unique_data_type, permission=access, delete_permission=update)

	assert helpers.result_names(await reset(nex.STRANGER, [data_id], False)) == [None]
	assert await helpers.rating_slot_info(data_id) == (10, 0, 10)

# * Object status tests
# *
# * Not even the owner can reset an object which is under review or
# * rejected
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

@pytest.mark.parametrize("status", [
	pytest.param(constants.DataStatus.DATA_STATUS_PENDING, id="pending"),
	pytest.param(constants.DataStatus.DATA_STATUS_REJECTED, id="rejected")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_the_owner_cannot_reset_an_object_which_is_not_available(unique_data_type, status, transactional):
	"""
	The owner is told the object does not exist, and its ratings are
	left alone
	"""

	data_id = await post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=status)

	if transactional:
		assert await reset_error(nex.FRIEND_A, [data_id], True) == "DataStore::NotFound"
	else:
		assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

	assert await helpers.rating_slot_info(data_id) == (15, 1, 10)

async def test_an_object_posted_needing_review_cannot_be_reset_by_its_owner(unique_data_type):
	"""
	The same goes for an object which was under review from the start
	"""

	data_id = await post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == ["DataStore::NotFound"]

@pytest.mark.parametrize("status", [
	pytest.param(constants.DataStatus.DATA_STATUS_PENDING, id="pending"),
	pytest.param(constants.DataStatus.DATA_STATUS_REJECTED, id="rejected")
])
async def test_an_object_which_is_not_available_is_under_review_for_everyone_else(unique_data_type, status):
	"""
	Everyone but the owner is told it is under review, whatever their
	update permission is
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await post_rated(data_type=unique_data_type, delete_permission=update)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=status)

	assert helpers.result_names(await reset(nex.STRANGER, [data_id], False)) == ["DataStore::UnderReviewing"]

async def test_read_locks_do_not_stop_a_reset(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == [None]

# * The rating slots settings

@pytest.mark.parametrize("lock", [
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERMANENT}, id="permanent"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_INTERVAL, "period_duration": 3600}, id="interval"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERIOD, "period_duration": constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON}, id="period")
])
async def test_a_reset_lifts_every_rating_lock_on_every_slot(unique_data_type, lock):
	"""
	The rating logs are removed along with the ratings, so every user
	locked out of any of the objects slots can rate it again
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, **lock), helpers.rating_slot(1, **lock)])

	for slot in [0, 1]:
		await helpers.rate_object(nex.FRIEND_A, data_id, 1, slot=slot)
		await helpers.rate_object(nex.STRANGER, data_id, 1, slot=slot)

	await reset(nex.FRIEND_A, [data_id], False)

	for slot in [0, 1]:
		assert await helpers.rate_object_result(nex.FRIEND_A, data_id, slot) is None
		assert await helpers.rate_object_result(nex.STRANGER, data_id, slot) is None

async def test_modifiable_starts_a_new_rating_after_a_reset(unique_data_type):
	"""
	The users old rating is gone, so rating again adds a new one rather
	than changing the old one
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE, initial_value=10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, [data_id], False)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 7)) == (17, 1, 10)

async def test_round_minus_goes_back_to_the_unrounded_initial_value(unique_data_type):
	"""
	A negative initial value which was rounded up to 0 by a rating is
	negative again after a reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=-10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 0)
	await reset(nex.FRIEND_A, [data_id], False)

	assert await helpers.rating_slot_info(data_id) == (-10, 0, -10)

async def test_a_reset_clears_round_minus_and_modifiable_ratings(unique_data_type):
	"""
	With RATING_FLAG_ROUND_MINUS and RATING_FLAG_MODIFIABLE, changing a
	rating takes the old one away as it was given. After a reset there is
	no old rating left, so the next one is simply added
	"""

	flag = constants.RatingFlag.RATING_FLAG_ROUND_MINUS | constants.RatingFlag.RATING_FLAG_MODIFIABLE
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=flag)])

	await helpers.rate_object(nex.FRIEND_A, data_id, -10)
	await helpers.rate_object(nex.STRANGER, data_id, 3)
	await reset(nex.FRIEND_A, [data_id], False)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 1)) == (1, 1, 0)

async def test_disable_self_rating_does_not_stop_the_owner_resetting(unique_data_type):
	"""
	The owner may not rate the slot, but can still reset it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	await helpers.rate_object(nex.STRANGER, data_id, 1)
	await reset(nex.FRIEND_A, [data_id], False)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_a_slot_which_cannot_be_rated_can_still_be_reset(unique_data_type):
	"""
	Some of the rating flag bits which no flag uses make a slot impossible
	to rate, but it can still be reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=0x01, initial_value=3)])

	assert helpers.result_names(await reset(nex.FRIEND_A, [data_id], False)) == [None]
	assert await helpers.rating_slot_info(data_id) == (3, 0, 3)

# * Resetting does not count as referring to the object

async def test_resetting_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it. Resetting its
	slots does not count as updating it either, so none of its times move
	"""

	data_id = await post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await reset(nex.FRIEND_A, [data_id], False)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.update_time.value() == before.update_time.value()
	assert after.expire_time.value() == before.expire_time.value()
