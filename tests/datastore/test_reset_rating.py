import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def reset(pid: int, data_id: int, slot: int = 0, update_password: int = constants.INVALID_PASSWORD) -> None:
	"""
	Resets one slot of an object as the given user
	"""

	async with helpers.connect(pid) as client:
		await client.reset_rating(helpers.rating_target(data_id, slot), update_password)

async def reset_error(pid: int, data_id: int, slot: int = 0, update_password: int = constants.INVALID_PASSWORD) -> str:
	"""
	Resets one slot of an object and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.reset_rating(helpers.rating_target(data_id, slot), update_password)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::ResetRating did not throw an error")

# * What a reset does

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_reset_takes_the_slot_back_to_its_initial_value(unique_data_type, post):
	"""
	Every users ratings are removed, so the total goes back to the
	initial value and nothing is counted
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(initial_value=10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await helpers.rate_object(nex.STRANGER, data_id, 3)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (10, 0, 10)

async def test_the_slot_can_be_rated_again_after_a_reset(unique_data_type):
	"""
	Ratings start counting again from the initial value
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 2)) == (12, 1, 10)

async def test_only_the_slot_asked_for_is_reset(unique_data_type):
	"""
	The objects other slots keep their ratings
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, initial_value=10), helpers.rating_slot(1, initial_value=20)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await helpers.rate_object(nex.FRIEND_A, data_id, 7, slot=1)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id, 0) == (10, 0, 10)
	assert await helpers.rating_slot_info(data_id, 1) == (27, 1, 20)

async def test_resetting_a_slot_which_was_never_rated(unique_data_type):
	"""
	Trying to reset a slot which has no ratings does nothing
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=4)])

	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (4, 0, 4)

# * target.data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to reset
	"""

	assert await reset_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await reset_error(nex.FRIEND_A, 999999) == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects have no ratings to reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await reset_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot()])

	assert await reset_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await reset_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

# * target.slot

async def test_the_last_slot(unique_data_type):
	"""
	Slot RATING_SLOT_MAX is a slot like any other
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(constants.RATING_SLOT_MAX)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5, slot=constants.RATING_SLOT_MAX)
	await reset(nex.FRIEND_A, data_id, slot=constants.RATING_SLOT_MAX)

	assert await helpers.rating_slot_info(data_id, constants.RATING_SLOT_MAX) == (0, 0, 0)

async def test_a_slot_which_was_never_initialised(unique_data_type):
	"""
	Only the slots the object was posted with can be reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await reset_error(nex.FRIEND_A, data_id, slot=1) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_RATING_SLOT, id="past-the-last"),
	pytest.param(-1, id="negative")
])
async def test_a_slot_which_does_not_exist(unique_data_type, slot):
	"""
	Only slots 0 to RATING_SLOT_MAX exist
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await reset_error(nex.FRIEND_A, data_id, slot=slot) == "DataStore::InvalidArgument"

# * update_password

async def test_update_password_replaces_the_permission(unique_data_type):
	"""
	Someone without update permission can still reset the slot with the
	objects update password
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.STRANGER, data_id, update_password=password_info.update_password)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_wrong_update_password(unique_data_type):
	"""
	A password which does not match is refused
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await reset_error(nex.STRANGER, data_id, update_password=1234) == "DataStore::InvalidPassword"

async def test_the_access_password_is_not_the_update_password(unique_data_type):
	"""
	Resets are guarded behind the update permission
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await reset_error(nex.STRANGER, data_id, update_password=password_info.access_password) == "DataStore::InvalidPassword"

async def test_wrong_update_password_as_the_owner(unique_data_type):
	"""
	The owner already has update permission, so the password is
	never checked
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id, update_password=1234)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * The objects update permission
# *
# * Resetting a slot is gated by the objects update permission

@pytest.mark.parametrize("permission, recipients, pid, allowed", helpers.UPDATE_PERMISSIONS)
async def test_the_update_permission(unique_data_type, permission, recipients, pid, allowed):
	"""
	Only users the update permission allows can reset the slot, and a
	refused reset leaves the ratings alone
	"""

	update = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_rated(data_type=unique_data_type, delete_permission=update)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	if not allowed:
		assert await reset_error(pid, data_id) == "DataStore::PermissionDenied"
		assert await helpers.rating_slot_info(data_id) == (5, 1, 0)
		return

	await reset(pid, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_the_access_permission_is_not_used(unique_data_type):
	"""
	Someone who may update an object can reset it even if they cannot
	read it
	"""

	access = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=access, delete_permission=update)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.STRANGER, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * Object status tests
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

async def test_a_pending_object_can_be_reset_by_its_owner(unique_data_type):
	"""
	The slots of an object under review can still be reset by its owner
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review, whatever their
	update permission is
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_rated(data_type=unique_data_type, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await reset_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_a_rejected_object_can_be_reset_by_its_owner(unique_data_type):
	"""
	The slots of a rejected object can still be reset by its owner
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Rejected objects appear as under review still to other users
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type, delete_permission=update)

	assert await reset_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_read_locks_do_not_stop_a_reset(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)
	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * The rating slots settings

@pytest.mark.parametrize("lock", [
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERMANENT}, id="permanent"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_INTERVAL, "period_duration": 3600}, id="interval"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERIOD, "period_duration": constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON}, id="period")
])
async def test_a_reset_lifts_every_rating_lock(unique_data_type, lock):
	"""
	The rating logs are removed along with the ratings, so every user
	locked out of the slot can rate it again
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(**lock)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 1)
	await helpers.rate_object(nex.STRANGER, data_id, 1)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rate_object_result(nex.FRIEND_A, data_id) is None
	assert await helpers.rate_object_result(nex.STRANGER, data_id) is None
	assert await helpers.rate_object_result(nex.STRANGER, data_id) == "DataStore::OperationNotAllowed"

async def test_the_locks_on_other_slots_are_kept(unique_data_type):
	"""
	Only the slot asked for has its logs removed
	"""

	lock_type = constants.RatingLockType.RATING_LOCK_PERMANENT
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, lock_type=lock_type), helpers.rating_slot(1, lock_type=lock_type)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 1)
	await helpers.rate_object(nex.FRIEND_A, data_id, 1, slot=1)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rate_object_result(nex.FRIEND_A, data_id, slot=1) == "DataStore::OperationNotAllowed"

async def test_modifiable_starts_a_new_rating_after_a_reset(unique_data_type):
	"""
	The users old rating is gone, so rating again adds a new one rather
	than changing the old one
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE, initial_value=10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await reset(nex.FRIEND_A, data_id)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 7)) == (17, 1, 10)

async def test_round_minus_goes_back_to_the_unrounded_initial_value(unique_data_type):
	"""
	A negative initial value which was rounded up to 0 by a rating is
	negative again after a reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=-10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 0)
	await reset(nex.FRIEND_A, data_id)

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
	await reset(nex.FRIEND_A, data_id)

	assert helpers.rating_totals(await helpers.rate_object(nex.FRIEND_A, data_id, 1)) == (1, 1, 0)

async def test_disable_self_rating_does_not_stop_the_owner_resetting(unique_data_type):
	"""
	The owner may not rate the slot, but can still reset it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	await helpers.rate_object(nex.STRANGER, data_id, 1)
	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_a_slot_which_cannot_be_rated_can_still_be_reset(unique_data_type):
	"""
	Some of the rating flag bits which no flag uses make a slot impossible
	to rate, but it can still be reset
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=0x01, initial_value=3)])

	await reset(nex.FRIEND_A, data_id)

	assert await helpers.rating_slot_info(data_id) == (3, 0, 3)

# * Resetting a slot does not count as referring to the object

async def test_resetting_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it. Resetting a slot
	does not count as updating it either, so none of its times move
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)

	await helpers.rate_object(nex.FRIEND_A, data_id, 1)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await reset(nex.FRIEND_A, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.update_time.value() == before.update_time.value()
	assert after.expire_time.value() == before.expire_time.value()
