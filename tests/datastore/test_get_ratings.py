import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

# * The most data IDs one request can have
CAPACITY = 512

async def get_ratings(pid: int, data_ids: list[int], access_password: int = constants.INVALID_PASSWORD):
	"""
	Gets every rating slot of several objects as the given user
	"""

	async with helpers.connect(pid) as client:
		return await client.get_ratings(data_ids, access_password)

async def get_ratings_error(pid: int, data_ids: list[int], access_password: int = constants.INVALID_PASSWORD) -> str:
	"""
	Gets every rating slot of several objects and expects an error to be
	thrown for the whole request, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_ratings(data_ids, access_password)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetRatings did not throw an error")

async def get_one(pid: int, data_id: int, access_password: int = constants.INVALID_PASSWORD):
	"""
	Gets every rating slot of a single object, returning its slots and
	the result for it
	"""

	response = await get_ratings(pid, [data_id], access_password)

	return slot_totals(response.ratings[0]), helpers.result_names(response.results)[0]

def slot_totals(slots: list) -> list[tuple[int, int, int, int]]:
	"""
	The slot, total, count and initial value of every slot of an object
	"""

	return [(slot.slot, slot.info.total_value, slot.info.count, slot.info.initial_value) for slot in slots]

# * What comes back

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_slot_of_the_object(unique_data_type, post):
	"""
	Every slot the object was posted with comes back, in slot order
	"""

	slots = [helpers.rating_slot(15, initial_value=150), helpers.rating_slot(0, initial_value=1), helpers.rating_slot(3, initial_value=30)]
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=slots)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5, slot=3)
	await helpers.rate_object(nex.STRANGER, data_id, 2, slot=3)

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 1, 0, 1), (3, 37, 2, 30), (15, 150, 0, 150)], None)

async def test_an_object_with_no_slots(unique_data_type):
	"""
	An object posted without any slots has nothing to give back
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[])

	assert await get_one(nex.FRIEND_A, data_id) == ([], None)

@pytest.mark.parametrize("initial_value", [
	pytest.param(helpers.INT64_MAX, id="int64-max"),
	pytest.param(helpers.INT64_MIN, id="int64-min")
])
async def test_the_largest_initial_values(unique_data_type, initial_value):
	"""
	The initial value and total are SInt64s
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=initial_value)])

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, initial_value, 0, initial_value)], None)

# * data_ids

async def test_the_entries_are_in_the_order_asked_for(unique_data_type):
	"""
	Every object gets an entry, in the order the data IDs were given
	"""

	first = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=1)])
	second = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=2)])
	response = await get_ratings(nex.FRIEND_A, [second, first])

	assert [slot_totals(slots) for slots in response.ratings] == [[(0, 2, 0, 2)], [(0, 1, 0, 1)]]

async def test_a_data_id_asked_for_twice_comes_back_twice(unique_data_type):
	"""
	Each data ID gets its own entry, even when it is the same object
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=1)])
	response = await get_ratings(nex.FRIEND_A, [data_id, data_id])

	assert [slot_totals(slots) for slots in response.ratings] == [[(0, 1, 0, 1)], [(0, 1, 0, 1)]]

async def test_no_data_ids():
	"""
	Asking for nothing gets an empty list
	"""

	response = await get_ratings(nex.FRIEND_A, [])

	assert response.ratings == []
	assert response.results == []

async def test_the_most_data_ids_allowed(unique_data_type):
	"""
	CAPACITY data IDs are allowed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	response = await get_ratings(nex.FRIEND_A, [data_id] * CAPACITY)

	assert len(response.ratings) == CAPACITY

async def test_too_many_data_ids(unique_data_type):
	"""
	More than CAPACITY data IDs fails the whole request
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await get_ratings_error(nex.FRIEND_A, [data_id] * (CAPACITY + 1)) == "DataStore::OverCapacity"

async def test_a_data_id_which_names_nothing(unique_data_type):
	"""
	An object which cannot be found only fails its own entry, which has
	no slots. The rest still come back
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	response = await get_ratings(nex.FRIEND_A, [data_id, helpers.UNKNOWN_DATA_ID])

	assert helpers.result_names(response.results) == [None, "DataStore::NotFound"]
	assert [slot_totals(slots) for slots in response.ratings] == [[(0, 0, 0, 0)], []]

async def test_a_data_id_of_0_fails_the_whole_request(unique_data_type):
	"""
	Unlike a data ID which names nothing, INVALID_DATAID anywhere in the
	list fails everything
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await get_ratings_error(nex.FRIEND_A, [data_id, constants.INVALID_DATAID]) == "DataStore::InvalidArgument"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await get_one(nex.FRIEND_A, 999999) == ([], "DataStore::NotFound")

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects have no ratings to read
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await get_one(nex.FRIEND_A, data_id) == ([], "DataStore::NotFound")

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot()])

	assert await get_one(nex.FRIEND_A, data_id) == ([], "DataStore::NotFound")

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await get_one(nex.FRIEND_A, data_id) == ([], "DataStore::NotFound")

# * access_password

async def test_no_access_password_without_permission(unique_data_type):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails its entry with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await get_one(nex.STRANGER, data_id) == ([], "DataStore::PermissionDenied")

async def test_wrong_access_password_without_permission(unique_data_type):
	"""
	Using the wrong password on an object the caller has no access
	to fails its entry with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await get_one(nex.STRANGER, data_id, access_password=1234) == ([], "DataStore::InvalidPassword")

async def test_wrong_access_password_with_permission(unique_data_type):
	"""
	Using the wrong password on an object the user already has access
	to is not an error
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await get_one(nex.STRANGER, data_id, access_password=1234) == ([(0, 0, 0, 0)], None)

async def test_access_password_bypasses_permissions(unique_data_type):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await get_one(nex.STRANGER, data_id, access_password=password_info.access_password) == ([(0, 0, 0, 0)], None)

async def test_the_update_password_is_not_the_access_password(unique_data_type):
	"""
	Reading is guarded by access permissions
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await get_one(nex.STRANGER, data_id, access_password=password_info.update_password) == ([], "DataStore::InvalidPassword")

async def test_one_access_password_for_every_entry(unique_data_type):
	"""
	The password only opens the object it belongs to
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	public = await helpers.post_rated(data_type=unique_data_type)
	first = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	second = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, first)
	response = await get_ratings(nex.STRANGER, [public, first, second], access_password=password_info.access_password)

	assert helpers.result_names(response.results) == [None, None, "DataStore::InvalidPassword"]

# * Access permission tests
# *
# * Objects can always be read by their owner. For everyone else, it
# * depends on the access permission of the object

@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, permission, recipients, viewer, visible):
	"""
	Only users who can see an object can read its ratings. An object the
	viewer cannot see only fails its own entry
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=access)

	if not visible:
		assert await get_one(viewer, data_id) == ([], "DataStore::PermissionDenied")
		return

	assert await get_one(viewer, data_id) == ([(0, 0, 0, 0)], None)

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

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 0, 0, 0)], None)

async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await get_one(nex.STRANGER, data_id) == ([], "DataStore::UnderReviewing")

async def test_a_rejected_object_can_be_read_by_its_owner(unique_data_type):
	"""
	The ratings of a rejected object can still be read by its owner
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 0, 0, 0)], None)

async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Rejected objects appear as under review still to other users
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_one(nex.STRANGER, data_id) == ([], "DataStore::UnderReviewing")

async def test_read_locks_do_not_stop_reading(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)

	assert await get_one(nex.STRANGER, data_id) == ([(0, 0, 0, 0)], None)

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

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 2, 1, 0)], None)

async def test_disable_self_rating_does_not_stop_the_owner_reading(unique_data_type):
	"""
	The owner may not rate the slot, but can still read it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 0, 0, 0)], None)

async def test_round_minus_only_rounds_once_the_slot_is_rated(unique_data_type):
	"""
	The total is only rounded up to 0 when the slot is rated, so a
	negative initial value is read as it is until then. Afterwards the
	rounded total is what is read, even when the rating itself was 0
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=-10)])

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, -10, 0, -10)], None)

	await helpers.rate_object(nex.FRIEND_A, data_id, 0)

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 0, 1, -10)], None)

async def test_slots_which_cannot_be_rated_can_still_be_read(unique_data_type):
	"""
	Some of the rating flag bits which no flag uses make a slot impossible
	to rate, but it can still be read
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, flag=0x01, initial_value=3), helpers.rating_slot(1, flag=0x02, initial_value=4)])

	assert await get_one(nex.FRIEND_A, data_id) == ([(0, 3, 0, 3), (1, 4, 0, 4)], None)

# * Reading the ratings does not count as referring to the object

async def test_reading_the_ratings_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await get_ratings(nex.STRANGER, [data_id])

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.expire_time.value() == before.expire_time.value()
