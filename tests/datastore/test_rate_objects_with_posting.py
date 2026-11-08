import time

import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * DISCLAIMER, THESE TESTS ASSUME THE WIKI IS CORRECT AND HAVE NOT BEEN
# * CHECKED AGAINST A REAL SERVER. THE ONLY LIVE NEX SERVER ON NINTENDO
# * NETWORK AT THE TIME OF WRITING IS POKEMON BANK, WHICH HAS CUSTOM RMCs
# * FROM 40+, SO THIS COULD NOT BE TESTED. ANYTHING THE WIKI DOES NOT COVER
# * IS ASSUMED TO WORK THE SAME AS RateObjectWithPosting

async def rate(pid: int, targets: list, params: list, transactional: bool = False, fetch_ratings: bool = True, post_params: list | None = None):
	"""
	Rates several slots as the given user, returning the ratings and
	results. Each target is a DataStoreRatingTarget, each param a
	DataStoreRateObjectParam, and each post param a
	DataStorePreparePostParam. Without post params, one which sets up no
	slots is used for every target
	"""

	async with helpers.connect(pid) as client:
		return await client.rate_objects_with_posting(targets, params, post_params or [helpers.post_param()], transactional, fetch_ratings)

async def rate_error(pid: int, targets: list, params: list, transactional: bool = False, post_params: list | None = None) -> str:
	"""
	Rates several slots and expects an error to be thrown for the whole
	request, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.rate_objects_with_posting(targets, params, post_params or [helpers.post_param()], transactional, True)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::RateObjectsWithPosting did not throw an error")

def rating(rating_value: int, access_password: int = constants.INVALID_PASSWORD) -> datastore.DataStoreRateObjectParam:
	"""
	The rating to give, and the password to give it with
	"""

	return helpers.rate_object_param(rating_value, access_password)

async def rate_one(pid: int, data_id: int, rating_value: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD, slots: list | None = None) -> datastore.DataStoreRatingInfo:
	"""
	Rates one slot on its own, expecting it to work, and returns what the
	slot holds afterwards. slots are the rating slot settings sent in the
	post param, which set up the slot if it was never initialised
	"""

	response = await rate(pid, [helpers.rating_target(data_id, slot)], [rating(rating_value, access_password)], post_params=[helpers.post_param(rating_init_param=slots or [])])

	assert helpers.result_names(response.results) == [None]

	return response.ratings[0]

async def rate_one_error(pid: int, data_id: int, rating_value: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD, slots: list | None = None) -> str:
	"""
	Rates one slot on its own and expects it to fail, returning the name
	of its result
	"""

	response = await rate(pid, [helpers.rating_target(data_id, slot)], [rating(rating_value, access_password)], post_params=[helpers.post_param(rating_init_param=slots or [])])
	[result] = helpers.result_names(response.results)

	if result is None:
		pytest.fail("DataStoreProtocol::RateObjectsWithPosting did not fail the rating")

	return result

async def rate_with(client, data_id: int, rating_value: int) -> str | None:
	"""
	Rates slot 0 over a connection which is already open, returning the
	name of the error, or None if it worked. Used where logging in again
	for every rating would throw the timing off
	"""

	response = await client.rate_objects_with_posting([helpers.rating_target(data_id, 0)], [rating(rating_value)], [helpers.post_param()], False, True)

	return helpers.result_names(response.results)[0]

# * What rating does

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_rating_adds_to_the_total(unique_data_type, post):
	"""
	The total starts at the initial value, and each rating adds to it and
	to the count. What the slot holds afterwards is returned
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(initial_value=10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 5)) == (15, 1, 10)
	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 3)) == (18, 2, 10)

async def test_everyones_ratings_add_up(unique_data_type):
	"""
	Each user adds a rating of their own to the same total
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await rate_one(nex.FRIEND_A, data_id, 5)
	await rate_one(nex.STRANGER, data_id, 3)

	assert await helpers.rating_slot_info(data_id) == (8, 2, 0)

async def test_each_slot_is_rated_on_its_own(unique_data_type):
	"""
	Rating one slot leaves the others alone
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, initial_value=1), helpers.rating_slot(1, initial_value=100)])

	await rate_one(nex.FRIEND_A, data_id, 5, slot=1)

	assert await helpers.rating_slot_info(data_id, 0) == (1, 0, 1)
	assert await helpers.rating_slot_info(data_id, 1) == (105, 1, 100)

async def test_every_target_is_rated(unique_data_type):
	"""
	Each target gets its own rating, and the ratings come back in the
	order the targets were given
	"""

	first = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, initial_value=10), helpers.rating_slot(2, initial_value=20)])
	second = await helpers.post_rated(data_type=unique_data_type)
	response = await rate(nex.FRIEND_A, [helpers.rating_target(first, 2), helpers.rating_target(second, 0)], [rating(5), rating(3)])

	assert helpers.result_names(response.results) == [None, None]
	assert [helpers.rating_totals(info) for info in response.ratings] == [(25, 1, 20), (3, 1, 0)]
	assert await helpers.rating_slot_info(first, 0) == (10, 0, 10)

async def test_the_same_slot_twice(unique_data_type):
	"""
	Each target is rated in turn, so the same slot can be rated more than
	once in a request
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	response = await rate(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * 2, [rating(5), rating(3)])

	assert helpers.result_names(response.results) == [None, None]
	assert await helpers.rating_slot_info(data_id) == (8, 2, 0)

# * targets

async def test_no_targets():
	"""
	Rating nothing does nothing
	"""

	response = await rate(nex.FRIEND_A, [], [])

	assert response.ratings == []
	assert response.results == []

@pytest.mark.parametrize("transactional", [True, False])
async def test_exactly_the_capacity(unique_data_type, transactional):
	"""
	BATCH_PROCESSING_CAPACITY_POST_OBJECT targets are allowed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await rate(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * constants.BATCH_PROCESSING_CAPACITY_POST_OBJECT, [rating(1)], transactional)

	assert await helpers.rating_slot_info(data_id) == (constants.BATCH_PROCESSING_CAPACITY_POST_OBJECT, constants.BATCH_PROCESSING_CAPACITY_POST_OBJECT, 0)

@pytest.mark.parametrize("transactional", [True, False])
async def test_too_many_targets(unique_data_type, transactional):
	"""
	More than BATCH_PROCESSING_CAPACITY_POST_OBJECT targets fails the whole
	request, and nothing is rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	count = constants.BATCH_PROCESSING_CAPACITY_POST_OBJECT + 1

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * count, [rating(1)] * count, transactional) == "DataStore::OverCapacity"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * targets, data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to rate
	"""

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::NotFound"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await rate_one_error(nex.FRIEND_A, 999999, 1) == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects cannot be rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot()])

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::NotFound"

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::NotFound"

# * targets, slot

async def test_the_last_slot(unique_data_type):
	"""
	Slot RATING_SLOT_MAX is a slot like any other
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(constants.RATING_SLOT_MAX)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1, slot=constants.RATING_SLOT_MAX)) == (1, 1, 0)

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_RATING_SLOT, id="past-the-last"),
	pytest.param(-1, id="negative")
])
async def test_a_slot_which_does_not_exist(unique_data_type, slot):
	"""
	Only slots 0 to RATING_SLOT_MAX exist, even when the post param has
	settings for it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1, slot=slot, slots=[helpers.rating_slot(slot)]) == "DataStore::InvalidArgument"

# * rate_params

async def test_one_param_for_every_target(unique_data_type):
	"""
	A single param is used for every target
	"""

	first = await helpers.post_rated(data_type=unique_data_type)
	second = await helpers.post_rated(data_type=unique_data_type)
	response = await rate(nex.FRIEND_A, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], [rating(4)])

	assert helpers.result_names(response.results) == [None, None]
	assert await helpers.rating_slot_info(first) == (4, 1, 0)
	assert await helpers.rating_slot_info(second) == (4, 1, 0)

@pytest.mark.parametrize("params", [
	pytest.param([], id="none"),
	pytest.param([rating(1), rating(2)], id="fewer-than-the-targets"),
	pytest.param([rating(1), rating(2), rating(3), rating(4)], id="more-than-the-targets")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_params_which_do_not_match_the_targets(unique_data_type, params, transactional):
	"""
	There must be one param, or one for each target. Anything else fails
	the whole request, and nothing is rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * 3, params, transactional) == "DataStore::InvalidArgument"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * rate_params, rating_value

async def test_a_rating_of_0_still_counts(unique_data_type):
	"""
	A rating of 0 leaves the total where it was, but is counted
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 0)) == (10, 1, 10)

async def test_a_negative_rating(unique_data_type):
	"""
	Ratings can be negative, and so can the total
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, -100)) == (-90, 1, 10)

@pytest.mark.parametrize("rating_value", [
	pytest.param(helpers.INT32_MAX, id="int32-max"),
	pytest.param(helpers.INT32_MIN, id="int32-min")
])
async def test_the_largest_ratings(unique_data_type, rating_value):
	"""
	Any SInt32 rating is allowed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, rating_value)) == (rating_value, 1, 0)

async def test_the_total_goes_past_what_a_rating_can_hold(unique_data_type):
	"""
	The total is an SInt64, so it can go past what one rating can be
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await rate_one(nex.FRIEND_A, data_id, helpers.INT32_MAX)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, helpers.INT32_MAX)) == (helpers.INT32_MAX * 2, 2, 0)

@pytest.mark.parametrize("initial_value, rating_value", [
	pytest.param(helpers.INT64_MAX - 5, 10, id="past-int64-max"),
	pytest.param(helpers.INT64_MIN + 5, -10, id="past-int64-min")
])
async def test_the_total_cannot_go_past_an_sint64(unique_data_type, initial_value, rating_value):
	"""
	A rating which would take the total past what an SInt64 holds crashes
	rather than wrapping around, and is not counted. A crash fails the
	whole request, not just the one rating
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=initial_value)])

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(data_id, 0)], [rating(rating_value)]) == "PythonCore::Exception"
	assert await helpers.rating_slot_info(data_id) == (initial_value, 0, initial_value)

# * rate_params, access_password

async def test_no_access_password_without_permission(unique_data_type):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await rate_one_error(nex.STRANGER, data_id, 1) == "DataStore::PermissionDenied"

async def test_wrong_access_password_without_permission(unique_data_type):
	"""
	Using the wrong password on an object the caller has no access
	to fails with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)

	assert await rate_one_error(nex.STRANGER, data_id, 1, access_password=1234) == "DataStore::InvalidPassword"

async def test_wrong_access_password_with_permission(unique_data_type):
	"""
	Using the wrong password on an object the user already has access
	to is not an error
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1, access_password=1234)) == (1, 1, 0)

async def test_access_password_bypasses_permissions(unique_data_type):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1, access_password=password_info.access_password)) == (1, 1, 0)

async def test_the_update_password_is_not_the_access_password(unique_data_type):
	"""
	Rating only needs access, and the update password does not give it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await rate_one_error(nex.STRANGER, data_id, 1, access_password=password_info.update_password) == "DataStore::InvalidPassword"

async def test_each_param_carries_its_own_access_password(unique_data_type):
	"""
	Each target is rated with the access password of its own param
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	first = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	second = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	first_password_info = await helpers.get_password_info(nex.FRIEND_A, first)
	second_password_info = await helpers.get_password_info(nex.FRIEND_A, second)
	params = [rating(1, first_password_info.access_password), rating(1, second_password_info.access_password)]
	response = await rate(nex.STRANGER, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], params)

	assert helpers.result_names(response.results) == [None, None]

async def test_one_param_shares_its_access_password(unique_data_type):
	"""
	A single param gives its access password to every target, so it only
	opens the object it belongs to
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	first = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	second = await helpers.post_rated(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, first)
	response = await rate(nex.STRANGER, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], [rating(1, password_info.access_password)])

	assert helpers.result_names(response.results) == [None, "DataStore::InvalidPassword"]

# * post_params
# *
# * A slot which was never initialised is set up from the post param
# * before it is rated. Only the rating slot settings of the post param
# * are used

async def test_a_slot_which_was_never_initialised_is_set_up(unique_data_type):
	"""
	The slot gets the settings the post param has for it, then is rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1, slot=1, slots=[helpers.rating_slot(1, initial_value=10)])) == (11, 1, 10)
	assert await helpers.rating_slot_info(data_id, 1) == (11, 1, 10)

async def test_an_object_with_no_slots_at_all(unique_data_type):
	"""
	An object posted without any rating slots can still have one set up
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1, slots=[helpers.rating_slot(initial_value=10)])) == (11, 1, 10)

async def test_a_slot_which_was_already_initialised_ignores_the_post_param(unique_data_type):
	"""
	The slot keeps the settings it was posted with
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1, slots=[helpers.rating_slot(initial_value=100)])) == (11, 1, 10)

async def test_the_slot_keeps_the_settings_from_the_post_param(unique_data_type):
	"""
	Every setting of the slot comes from the post param, not just the
	initial value, and they stay for later ratings
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await rate_one(nex.FRIEND_A, data_id, 1, slot=1, slots=[helpers.rating_slot(1, lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	assert await rate_one_error(nex.FRIEND_A, data_id, 1, slot=1) == "DataStore::OperationNotAllowed"

async def test_a_post_param_without_settings_for_the_slot(unique_data_type):
	"""
	There is nothing to set the slot up with, so it cannot be rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1, slot=1, slots=[helpers.rating_slot(2)]) == "DataStore::InvalidArgument"

async def test_only_the_slot_being_rated_is_set_up(unique_data_type):
	"""
	Settings in the post param for other slots are not used
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	await rate_one(nex.FRIEND_A, data_id, 1, slot=1, slots=[helpers.rating_slot(1), helpers.rating_slot(2)])

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)

	assert [rating.slot for rating in meta_info.ratings] == [0, 1]

async def test_only_the_rating_slot_settings_are_used(unique_data_type):
	"""
	The rest of the post param leaves the object alone
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, name="before")
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await rate(nex.FRIEND_A, [helpers.rating_target(data_id, 1)], [rating(1)], post_params=[helpers.post_param(name="after", data_type=unique_data_type + 1, period=30, rating_init_param=[helpers.rating_slot(1)])])

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.name == before.name
	assert after.data_type == before.data_type
	assert after.period == before.period

async def test_the_post_param_does_not_create_the_object():
	"""
	An object which does not exist is not posted, and cannot be rated
	"""

	assert await rate_one_error(nex.FRIEND_A, helpers.UNKNOWN_DATA_ID, 1, slots=[helpers.rating_slot()]) == "DataStore::NotFound"

async def test_one_post_param_for_every_target(unique_data_type):
	"""
	A single post param is used to set up every target
	"""

	first = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	second = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	response = await rate(nex.FRIEND_A, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], [rating(1)], post_params=[helpers.post_param(rating_init_param=[helpers.rating_slot(initial_value=10)])])

	assert helpers.result_names(response.results) == [None, None]
	assert await helpers.rating_slot_info(first) == (11, 1, 10)
	assert await helpers.rating_slot_info(second) == (11, 1, 10)

async def test_a_post_param_for_each_target(unique_data_type):
	"""
	Each target is set up from its own post param
	"""

	first = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	second = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	post_params = [helpers.post_param(rating_init_param=[helpers.rating_slot(initial_value=10)]), helpers.post_param(rating_init_param=[helpers.rating_slot(initial_value=20)])]
	response = await rate(nex.FRIEND_A, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], [rating(1)], post_params=post_params)

	assert helpers.result_names(response.results) == [None, None]
	assert await helpers.rating_slot_info(first) == (11, 1, 10)
	assert await helpers.rating_slot_info(second) == (21, 1, 20)

@pytest.mark.parametrize("count", [
	pytest.param(2, id="fewer-than-the-targets"),
	pytest.param(4, id="more-than-the-targets")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_post_params_which_do_not_match_the_targets(unique_data_type, count, transactional):
	"""
	There must be one post param, or one for each target. Anything else
	fails the whole request, and nothing is rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * 3, [rating(1)], transactional, post_params=[helpers.post_param()] * count) == "DataStore::InvalidArgument"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * transactional, set to true
# *
# * A transactional request is all or nothing. It throws the first error
# * it runs into, leaves every slot alone, and has no per-target results to
# * give back

async def test_transactional_rates_every_target(unique_data_type):
	"""
	Every target is rated, and there are no results because nothing can
	have gone wrong
	"""

	first = await helpers.post_rated(data_type=unique_data_type)
	second = await helpers.post_rated(data_type=unique_data_type)
	response = await rate(nex.FRIEND_A, [helpers.rating_target(first, 0), helpers.rating_target(second, 0)], [rating(1)], True)

	assert response.results == []
	assert await helpers.rating_slot_info(first) == (1, 1, 0)
	assert await helpers.rating_slot_info(second) == (1, 1, 0)

@pytest.mark.parametrize("position", [
	pytest.param(0, id="the-failure-comes-first"),
	pytest.param(1, id="the-failure-comes-last")
])
async def test_transactional_leaves_everything_alone_when_one_fails(unique_data_type, position):
	"""
	One target which cannot be rated fails the whole request, rolling back
	any changes, regardless of where the failure happened
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type)
	targets = [helpers.rating_target(data_id, 0), helpers.rating_target(helpers.UNKNOWN_DATA_ID, 0)]

	if position == 0:
		targets.reverse()

	assert await rate_error(nex.FRIEND_A, targets, [rating(1)], True) == "DataStore::NotFound"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_transactional_rolls_back_a_permission_failure(unique_data_type):
	"""
	A target the caller may not rate fails the request, and the other
	target in the same list is left alone
	"""

	public = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type)
	private = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type, permission=helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE))

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(public, 0), helpers.rating_target(private, 0)], [rating(1)], True) == "DataStore::PermissionDenied"
	assert await helpers.rating_slot_info(public) == (0, 0, 0)

@pytest.mark.parametrize("order, error", [
	pytest.param("permission-first", "DataStore::PermissionDenied", id="permission-first"),
	pytest.param("missing-first", "DataStore::NotFound", id="missing-first")
])
async def test_transactional_throws_the_first_error(unique_data_type, order, error):
	"""
	When more than one target fails, the error is the one for whichever
	comes first in the list
	"""

	private = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type, permission=helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE))
	targets = [helpers.rating_target(private, 0), helpers.rating_target(helpers.UNKNOWN_DATA_ID, 0)]

	if order == "missing-first":
		targets.reverse()

	assert await rate_error(nex.FRIEND_A, targets, [rating(1)], True) == error

async def test_transactional_rolls_back_a_lock_set_earlier_in_the_request(unique_data_type):
	"""
	Rating a locked slot twice fails on the second rating, and the first
	rating is rolled back with it
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	assert await rate_error(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * 2, [rating(1)], True) == "DataStore::OperationNotAllowed"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

# * transactional, set to false
# *
# * Without the transactional flag each target is dealt with on its own,
# * the request always succeeds, and the results list says what happened
# * to each one in turn

async def test_rates_what_it_can(unique_data_type):
	"""
	The targets which can be rated are rated
	"""

	first = await helpers.post_rated(data_type=unique_data_type)
	second = await helpers.post_rated(data_type=unique_data_type)
	targets = [helpers.rating_target(first, 0), helpers.rating_target(helpers.UNKNOWN_DATA_ID, 0), helpers.rating_target(second, 0)]
	response = await rate(nex.FRIEND_A, targets, [rating(1)])

	assert helpers.result_names(response.results) == [None, "DataStore::NotFound", None]
	assert await helpers.rating_slot_info(first) == (1, 1, 0)
	assert await helpers.rating_slot_info(second) == (1, 1, 0)

async def test_a_permission_failure_only_fails_its_own_entry(unique_data_type):
	"""
	A target the caller may not rate is left alone, and the other target
	is still rated
	"""

	public = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type)
	private = await helpers.post_rated(nex.STRANGER, data_type=unique_data_type, permission=helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE))
	response = await rate(nex.FRIEND_A, [helpers.rating_target(public, 0), helpers.rating_target(private, 0)], [rating(1)])

	assert helpers.result_names(response.results) == [None, "DataStore::PermissionDenied"]
	assert await helpers.rating_slot_info(public) == (1, 1, 0)

async def test_a_failed_entry_has_an_empty_rating(unique_data_type):
	"""
	Every target still gets an entry in the ratings, so a target which
	failed gets one which is zeroed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])
	response = await rate(nex.FRIEND_A, [helpers.rating_target(helpers.UNKNOWN_DATA_ID, 0), helpers.rating_target(data_id, 0)], [rating(1)])

	assert [helpers.rating_totals(info) for info in response.ratings] == [(0, 0, 0), (11, 1, 10)]

async def test_a_lock_set_earlier_in_the_request(unique_data_type):
	"""
	Rating a locked slot twice only fails the second rating
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])
	response = await rate(nex.FRIEND_A, [helpers.rating_target(data_id, 0)] * 2, [rating(1)])

	assert helpers.result_names(response.results) == [None, "DataStore::OperationNotAllowed"]
	assert await helpers.rating_slot_info(data_id) == (1, 1, 0)

# * fetch_ratings

@pytest.mark.parametrize("fetch_ratings", [
	pytest.param(True, id="true"),
	pytest.param(False, id="false")
])
async def test_fetch_ratings(unique_data_type, fetch_ratings):
	"""
	The ratings only come back when they are asked for. The slots are
	rated either way
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(initial_value=10)])
	response = await rate(nex.FRIEND_A, [helpers.rating_target(data_id, 0)], [rating(5)], fetch_ratings=fetch_ratings)

	if fetch_ratings:
		assert [helpers.rating_totals(info) for info in response.ratings] == [(15, 1, 10)]
	else:
		assert response.ratings == []

	assert await helpers.rating_slot_info(data_id) == (15, 1, 10)

# * Access permission tests
# *
# * Objects can always be rated by their owner. For everyone else, it
# * depends on the access permission of the object

@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, permission, recipients, viewer, visible):
	"""
	Only users who can see an object can rate it
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_rated(data_type=unique_data_type, permission=access)

	if not visible:
		assert await rate_one_error(viewer, data_id, 1) == "DataStore::PermissionDenied"
		return

	assert helpers.rating_totals(await rate_one(viewer, data_id, 1)) == (1, 1, 0)

# * Object status tests
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

async def test_a_pending_object_can_be_rated_by_its_owner(unique_data_type):
	"""
	Objects under review can still be rated by their owner
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1)) == (1, 1, 0)

async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await rate_one_error(nex.STRANGER, data_id, 1) == "DataStore::UnderReviewing"

async def test_a_rejected_object_can_be_rated_by_its_owner(unique_data_type):
	"""
	Rejected objects can still be rated by their owner
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1)) == (1, 1, 0)

async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Rejected objects appear as under review still to other users
	"""

	data_id = await helpers.post_rejected(helpers.post_rated, nex.FRIEND_A, data_type=unique_data_type)

	assert await rate_one_error(nex.STRANGER, data_id, 1) == "DataStore::UnderReviewing"

async def test_read_locks_do_not_stop_rating(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1)) == (1, 1, 0)

# * The rating slot, RATING_FLAG_MODIFIABLE

async def test_modifiable_replaces_the_users_own_rating(unique_data_type):
	"""
	Rating again changes the users rating rather than adding another,
	so the count stays the same
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE, initial_value=10)])

	await rate_one(nex.FRIEND_A, data_id, 5)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 7)) == (17, 1, 10)

async def test_modifiable_keeps_one_rating_per_user(unique_data_type):
	"""
	Changing one users rating leaves everyone elses alone
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE)])

	await rate_one(nex.FRIEND_A, data_id, 5)
	await rate_one(nex.STRANGER, data_id, 3)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1)) == (4, 2, 0)

# * The rating slot, RATING_FLAG_ROUND_MINUS

async def test_round_minus_keeps_the_total_at_0_or_above(unique_data_type):
	"""
	A rating which would take the total below 0 leaves it at 0. The
	rating itself is not rounded
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, -5)) == (5, 1, 10)
	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, -20)) == (0, 2, 10)
	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 3)) == (3, 3, 10)

async def test_round_minus_with_a_negative_initial_value(unique_data_type):
	"""
	The total is rounded up to 0 as soon as anything is rated, even a
	rating of 0
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_ROUND_MINUS, initial_value=-10)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 0)) == (0, 1, -10)

async def test_round_minus_with_modifiable_takes_away_the_unrounded_rating(unique_data_type):
	"""
	Changing a rating takes the old rating away from the total and adds
	the new one. The old rating is taken away as it was given, even when
	the total was rounded up to 0 at the time, so the total can end up
	higher than every rating added together (basically `3 - (-10) + 1`)
	"""

	flag = constants.RatingFlag.RATING_FLAG_ROUND_MINUS | constants.RatingFlag.RATING_FLAG_MODIFIABLE
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=flag)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, -10)) == (0, 1, 0)
	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 3)) == (3, 2, 0)
	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1)) == (14, 2, 0)

# * The rating slot, RATING_FLAG_DISABLE_SELF_RATING

async def test_disable_self_rating_refuses_the_owner(unique_data_type):
	"""
	The owner cannot rate their own object, and the rating is not counted
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::OperationNotAllowed"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

async def test_disable_self_rating_lets_everyone_else_rate(unique_data_type):
	"""
	Only the owner is refused
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING)])

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1)) == (1, 1, 0)

# * The rating slot, flags which are not defined
# *
# * The object can be posted with any flag, but some of the bits which no
# * flag uses make the slot impossible to rate

@pytest.mark.parametrize("flag, error", [
	pytest.param(0x01, "PythonCore::Exception", id="0x01"),
	pytest.param(0x02, "DataStore::InvalidArgument", id="0x02"),
	pytest.param(0x20, "DataStore::InvalidArgument", id="0x20"),
	pytest.param(0x40, None, id="0x40"),
	pytest.param(0x80, None, id="0x80")
])
async def test_a_rating_flag_which_is_not_defined(unique_data_type, flag, error):
	"""
	0x01, 0x02 and 0x20 refuse every rating, while 0x40 and 0x80 are
	ignored
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=flag)])

	if error == "PythonCore::Exception":
		# * A crash fails the whole request, not just the one rating
		assert await rate_error(nex.STRANGER, [helpers.rating_target(data_id, 0)], [rating(1)]) == error
		assert await helpers.rating_slot_info(data_id) == (0, 0, 0)
		return

	if error is not None:
		assert await rate_one_error(nex.STRANGER, data_id, 1) == error
		assert await helpers.rating_slot_info(data_id) == (0, 0, 0)
		return

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1)) == (1, 1, 0)

# * The rating slot, the range

USE_RANGE = constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MIN | constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MAX

@pytest.mark.parametrize("rating_value", [
	pytest.param(1, id="the-minimum"),
	pytest.param(5, id="the-maximum")
])
async def test_a_rating_inside_the_range(unique_data_type, rating_value):
	"""
	Both ends of the range are allowed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(internal_flag=USE_RANGE, range_min=1, range_max=5)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, rating_value)) == (rating_value, 1, 0)

@pytest.mark.parametrize("rating_value", [
	pytest.param(0, id="below"),
	pytest.param(6, id="above")
])
async def test_a_rating_outside_the_range(unique_data_type, rating_value):
	"""
	A rating outside the range is refused and not counted
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(internal_flag=USE_RANGE, range_min=1, range_max=5)])

	assert await rate_one_error(nex.FRIEND_A, data_id, rating_value) == "DataStore::InvalidArgument"
	assert await helpers.rating_slot_info(data_id) == (0, 0, 0)

@pytest.mark.parametrize("internal_flag, rating_value", [
	pytest.param(constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MIN, 100, id="only-the-minimum"),
	pytest.param(constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MAX, -100, id="only-the-maximum")
])
async def test_only_the_flagged_end_of_the_range_is_checked(unique_data_type, internal_flag, rating_value):
	"""
	Each end of the range has its own flag, and the other end is not
	checked at all
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(internal_flag=internal_flag, range_min=1, range_max=5)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, rating_value)) == (rating_value, 1, 0)

@pytest.mark.parametrize("internal_flag", [
	pytest.param(0, id="no-internal-flags"),
	pytest.param(0x01 | 0x08 | 0x80, id="internal-flags-which-are-not-defined")
])
async def test_a_range_without_its_flags_is_not_checked(unique_data_type, internal_flag):
	"""
	The range is only checked when the flags for it are set
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(internal_flag=internal_flag, range_min=1, range_max=5)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 100)) == (100, 1, 0)

async def test_the_initial_value_is_not_checked_against_the_range(unique_data_type):
	"""
	Only ratings are checked, the total can start outside the range
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(internal_flag=USE_RANGE, range_min=1, range_max=5, initial_value=100)])

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 3)) == (103, 1, 100)

async def test_a_change_outside_the_range_keeps_the_old_rating(unique_data_type):
	"""
	With RATING_FLAG_MODIFIABLE, a refused change leaves the users
	rating as it was
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE, internal_flag=USE_RANGE, range_min=1, range_max=5)])

	await rate_one(nex.FRIEND_A, data_id, 5)

	assert await rate_one_error(nex.FRIEND_A, data_id, 6) == "DataStore::InvalidArgument"
	assert await helpers.rating_slot_info(data_id) == (5, 1, 0)

# * The rating slot, the lock
# *
# * A lock stops a user rating a slot again until it lifts. Each user has
# * their own lock on each slot

async def test_a_permanent_lock(unique_data_type):
	"""
	The user can never rate the slot again
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	await rate_one(nex.FRIEND_A, data_id, 1)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::OperationNotAllowed"
	assert await helpers.rating_slot_info(data_id) == (1, 1, 0)

async def test_a_lock_is_per_user(unique_data_type):
	"""
	One user being locked does not stop anyone else
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	await rate_one(nex.FRIEND_A, data_id, 1)

	assert helpers.rating_totals(await rate_one(nex.STRANGER, data_id, 1)) == (2, 2, 0)
	assert await rate_one_error(nex.STRANGER, data_id, 1) == "DataStore::OperationNotAllowed"

async def test_a_lock_is_per_slot(unique_data_type):
	"""
	Being locked out of one slot does not stop the user rating another
	"""

	lock_type = constants.RatingLockType.RATING_LOCK_PERMANENT
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(0, lock_type=lock_type), helpers.rating_slot(1, lock_type=lock_type)])

	await rate_one(nex.FRIEND_A, data_id, 1)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1, slot=1)) == (1, 1, 0)

async def test_an_interval_lock_lifts_after_its_duration(unique_data_type):
	"""
	period_duration is how many seconds the lock lasts for
	"""

	duration = 3
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_INTERVAL, period_duration=duration)])

	# * One connection for the whole test, so logging in again does not
	# * eat into the lock
	async with helpers.connect(nex.FRIEND_A) as client:
		assert await rate_with(client, data_id, 1) is None
		assert await rate_with(client, data_id, 1) == "DataStore::OperationNotAllowed"

		await anyio.sleep(duration + 1)

		assert await rate_with(client, data_id, 1) is None

async def test_a_refused_rating_does_not_extend_an_interval_lock(unique_data_type):
	"""
	Future tries while locked does not extend the lock time
	"""

	duration = 6
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_INTERVAL, period_duration=duration)])

	async with helpers.connect(nex.FRIEND_A) as client:
		start = time.monotonic()

		assert await rate_with(client, data_id, 1) is None

		await anyio.sleep(duration / 2)

		assert await rate_with(client, data_id, 1) == "DataStore::OperationNotAllowed"

		# * Had the refused rating set the lock again, it would last until
		# * well past this point
		await anyio.sleep(duration + 1 - (time.monotonic() - start))

		assert await rate_with(client, data_id, 1) is None

async def test_an_interval_lock_of_0_seconds(unique_data_type):
	"""
	A lock which lasts no time at all never stops anyone
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_INTERVAL, period_duration=0)])

	await rate_one(nex.FRIEND_A, data_id, 1)

	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 1)) == (2, 2, 0)

async def test_a_period_lock(unique_data_type):
	"""
	The lock lasts until the next of the given day, at the given hour
	"""

	lock = helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERIOD, period_duration=constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, period_hour=0)
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[lock])

	await rate_one(nex.FRIEND_A, data_id, 1)

	assert await rate_one_error(nex.FRIEND_A, data_id, 1) == "DataStore::OperationNotAllowed"

async def test_modifiable_does_not_get_around_a_lock(unique_data_type):
	"""
	A locked user cannot change their rating either
	"""

	lock = helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE, lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[lock])

	await rate_one(nex.FRIEND_A, data_id, 5)

	assert await rate_one_error(nex.FRIEND_A, data_id, 7) == "DataStore::OperationNotAllowed"
	assert await helpers.rating_slot_info(data_id) == (5, 1, 0)

async def test_a_refused_rating_does_not_lock(unique_data_type):
	"""
	Only a rating which is counted sets the lock
	"""

	lock = helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT, internal_flag=constants.RatingInternalFlag.RATING_INTERNAL_FLAG_USE_RANGE_MAX, range_max=5)
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[lock])

	assert await rate_one_error(nex.FRIEND_A, data_id, 6) == "DataStore::InvalidArgument"
	assert helpers.rating_totals(await rate_one(nex.FRIEND_A, data_id, 5)) == (5, 1, 0)

# * Rating does not count as referring to the object

async def test_rating_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it. Rating does not
	count as updating it either, so none of its times move
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await rate_one(nex.STRANGER, data_id, 1)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.update_time.value() == before.update_time.value()
	assert after.expire_time.value() == before.expire_time.value()
