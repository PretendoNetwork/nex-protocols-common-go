import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

async def get_rating(pid: int, data_id: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD) -> datastore.DataStoreRatingInfo:
	"""
	Gets what one slot of an object holds, as the given user
	"""

	async with helpers.connect(pid) as client:
		return await client.get_rating(helpers.rating_target(data_id, slot), access_password)

async def get_rating_error(pid: int, data_id: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD) -> str:
	"""
	Gets what one slot of an object holds and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_rating(helpers.rating_target(data_id, slot), access_password)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetRating did not throw an error")

# * What comes back

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_slot_which_was_never_rated(unique_data_type, post):
	"""
	The total starts at the initial value
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(initial_value=10)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (10, 0, 10)

async def test_a_rated_slot(unique_data_type):
	"""
	General rating rules (total value, count, etc.) work
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await helpers.rate_object(nex.STRANGER, data_id, 3)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (18, 2, 10)

async def test_each_slot_is_read_on_its_own(unique_data_type):
	"""
	Only the slot asked for is read
	"""

	slots = [helpers.rating_slot(0, initial_value=10), helpers.rating_slot(1, initial_value=-7)]
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=slots)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id, slot=1)) == (-7, 0, -7)

@pytest.mark.parametrize("initial_value", [
	pytest.param(helpers.INT64_MAX, id="int64-max"),
	pytest.param(helpers.INT64_MIN, id="int64-min")
])
async def test_the_largest_initial_values(unique_data_type, initial_value):
	"""
	The initial value and total are SInt64s
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=initial_value)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (initial_value, 0, initial_value)

# * target.data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to read
	"""

	assert await get_rating_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await get_rating_error(nex.FRIEND_A, 999999) == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects have no ratings to read
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await get_rating_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot()])

	assert await get_rating_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await get_rating_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

# * target.slot

async def test_the_last_slot(unique_data_type):
	"""
	Slot RATING_SLOT_MAX is a slot like any other
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(constants.RATING_SLOT_MAX, initial_value=4)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id, slot=constants.RATING_SLOT_MAX)) == (4, 0, 4)

async def test_a_slot_which_was_never_initialised(unique_data_type):
	"""
	Only the slots the object was posted with can be read
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await get_rating_error(nex.FRIEND_A, data_id, slot=1) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_RATING_SLOT, id="past-the-last"),
	pytest.param(-1, id="negative")
])
async def test_a_slot_which_does_not_exist(unique_data_type, slot):
	"""
	Only slots 0 to RATING_SLOT_MAX exist
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await get_rating_error(nex.FRIEND_A, data_id, slot=slot) == "DataStore::InvalidArgument"

# * access_password

async def test_no_access_password_without_permission(unique_data_type):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await get_rating_error(nex.STRANGER, data_id) == "DataStore::PermissionDenied"

async def test_wrong_access_password_without_permission(unique_data_type):
	"""
	Using the wrong password on an object the caller has no access
	to fails with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await get_rating_error(nex.STRANGER, data_id, access_password=1234) == "DataStore::InvalidPassword"

async def test_wrong_access_password_with_permission(unique_data_type):
	"""
	Using the wrong password on an object the user already has access
	to is not an error
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert helpers.rating_totals(await get_rating(nex.STRANGER, data_id, access_password=1234)) == (0, 0, 0)

async def test_access_password_bypasses_permissions(unique_data_type):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert helpers.rating_totals(await get_rating(nex.STRANGER, data_id, access_password=password_info.access_password)) == (0, 0, 0)

async def test_the_update_password_is_not_the_access_password(unique_data_type):
	"""
	Reading guarded by the access permissions
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await get_rating_error(nex.STRANGER, data_id, access_password=password_info.update_password) == "DataStore::InvalidPassword"

# * Access permission tests
# *
# * Objects can always be read by their owner. For everyone else, it
# * depends on the access permission of the object

@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, permission, recipients, viewer, visible):
	"""
	Only users who can see an object can read its ratings
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=access)

	if not visible:
		assert await get_rating_error(viewer, data_id) == "DataStore::PermissionDenied"
		return

	assert helpers.rating_totals(await get_rating(viewer, data_id)) == (0, 0, 0)

# * Object status tests
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

async def test_a_pending_object_can_be_read_by_its_owner(unique_data_type):
	"""
	The ratings of an object under review can still be read by its owner
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (0, 0, 0)

async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await get_rating_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_a_rejected_object_can_be_read_by_its_owner(unique_data_type):
	"""
	The ratings of a rejected object can still be read by its owner
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (0, 0, 0)

async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Rejected objects appear as under review still to other users
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_rating_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_read_locks_do_not_stop_reading(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)

	assert helpers.rating_totals(await get_rating(nex.STRANGER, data_id)) == (0, 0, 0)

# * The rating slots settings
# *
# * They only decide who may rate the slot and how, so none of them stop
# * anyone reading it

async def test_a_user_locked_out_of_the_slot_can_still_read_it(unique_data_type):
	"""
	A rating lock only stops the user rating again
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 2)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (2, 1, 0)

async def test_disable_self_rating_does_not_stop_the_owner_reading(unique_data_type):
	"""
	The owner may not rate the slot, but can still read it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (0, 0, 0)

async def test_round_minus_only_rounds_once_the_slot_is_rated(unique_data_type):
	"""
	The total is only rounded up to 0 when the slot is rated, so a
	negative initial value is read as it is until then. Afterwards the
	rounded total is what is read, even when the rating itself was 0
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=-10)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (-10, 0, -10)

	await helpers.rate_object(nex.FRIEND_A, data_id, 0)

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (0, 1, -10)

@pytest.mark.parametrize("flag", [
	pytest.param(0x01, id="0x01"),
	pytest.param(0x02, id="0x02")
])
async def test_a_slot_which_cannot_be_rated_can_still_be_read(unique_data_type, flag):
	"""
	Some of the rating flag bits which no flag uses make a slot impossible
	to rate, but it can still be read
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=flag, initial_value=3)])

	assert helpers.rating_totals(await get_rating(nex.FRIEND_A, data_id)) == (3, 0, 3)

# * Reading the ratings does not count as referring to the object

async def test_reading_the_ratings_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await get_rating(nex.STRANGER, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.expire_time.value() == before.expire_time.value()
