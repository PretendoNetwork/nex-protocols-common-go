import datetime

import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def get_rating_with_log(pid: int, data_id: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD):
	"""
	Gets what one slot of an object holds, and the users own rating log
	for it, as the given user
	"""

	async with helpers.connect(pid) as client:
		return await client.get_rating_with_log(helpers.rating_target(data_id, slot), access_password)

async def get_rating_with_log_error(pid: int, data_id: int, slot: int = 0, access_password: int = constants.INVALID_PASSWORD) -> str:
	"""
	Gets a slot and the users rating log for it, and expects an error to
	be thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_rating_with_log(helpers.rating_target(data_id, slot), access_password)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetRatingWithLog did not throw an error")

async def post_logged(pid: int = nex.FRIEND_A, **fields) -> int:
	"""
	Posts an object whose slot 0 keeps a rating log
	"""

	return await helpers.post_rated(pid, slots=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE)], **fields)

def log_fields(response) -> tuple[bool, int, int, int]:
	"""
	Whether the user has rated the slot, who the log is for, their rating,
	and when their lock expires
	"""

	log = response.log

	return log.is_rated, log.pid, log.rating_value, log.lock_expiration_time.value()

def unrated(pid: int) -> tuple[bool, int, int, int]:
	"""
	The log of a user who has not rated the slot. The rating is -1 and the
	lock expiration time is a DateTime of 0
	"""

	return False, pid, -1, 0

async def rate_and_log(slot_fields: dict, unique_data_type: int) -> tuple[datetime.datetime, datetime.datetime]:
	"""
	Posts an object with a slot set up as given, rates it as FRIEND_A, and
	returns the servers time just before the rating along with when the
	lock expires
	"""

	async with helpers.connect(nex.FRIEND_A) as client:
		data_id = await client.post_meta_binary(helpers.post_param(data_type=unique_data_type, rating_init_param=[helpers.rating_slot(**slot_fields)]))
		now = (await client.get_meta(helpers.get_meta_param(data_id=data_id))).create_time.standard_datetime()

		await client.rate_object(helpers.rating_target(data_id, 0), helpers.rate_object_param(1, constants.INVALID_PASSWORD), True)

		response = await client.get_rating_with_log(helpers.rating_target(data_id, 0), constants.INVALID_PASSWORD)

	return now, response.log.lock_expiration_time.standard_datetime()

def next_weekday_at(now: datetime.datetime, period_duration: int, period_hour: int) -> datetime.datetime:
	"""
	The next time it is the given RatingLockPeriod day at the given hour
	"""

	weekday = -period_duration - 1
	midnight = now.replace(hour=0, minute=0, second=0, microsecond=0)

	for days in range(15):
		day = midnight + datetime.timedelta(days=days)

		if day.weekday() != weekday:
			continue

		candidate = day + datetime.timedelta(hours=period_hour)

		if candidate > now:
			return candidate

	raise AssertionError("no day found")

def days_after_the_next_hour(now: datetime.datetime, period_duration: int, period_hour: int) -> datetime.datetime:
	"""
	The next time it is the given hour, then the given number of days after
	"""

	candidate = now.replace(hour=period_hour, minute=0, second=0, microsecond=0)

	if candidate <= now:
		candidate += datetime.timedelta(days=1)

	return candidate + datetime.timedelta(days=period_duration)

# * What comes back

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_rating_is_what_the_slot_holds(unique_data_type, post):
	"""
	The rating is the same as GetRating would give
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(initial_value=10, flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await helpers.rate_object(nex.STRANGER, data_id, 3)

	assert helpers.rating_totals((await get_rating_with_log(nex.FRIEND_A, data_id)).rating) == (18, 2, 10)

async def test_the_log_of_a_user_who_has_not_rated(unique_data_type):
	"""
	Returns an empty log for when the user hasn't rated the slot
	"""

	data_id = await post_logged(data_type=unique_data_type)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

async def test_the_log_of_a_user_who_has_rated(unique_data_type):
	"""
	The log has the users rating, and no lock expiration time without a lock
	"""

	data_id = await post_logged(data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == (True, nex.FRIEND_A, 5, 0)

async def test_the_log_has_the_users_latest_rating(unique_data_type):
	"""
	A rating which was changed with RATING_FLAG_MODIFIABLE has the new
	value in the log
	"""

	data_id = await post_logged(data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)
	await helpers.rate_object(nex.FRIEND_A, data_id, 7)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == (True, nex.FRIEND_A, 7, 0)

async def test_each_user_has_their_own_log(unique_data_type):
	"""
	The log is always the callers
	"""

	data_id = await post_logged(data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	assert log_fields(await get_rating_with_log(nex.STRANGER, data_id)) == unrated(nex.STRANGER)

async def test_a_reset_clears_the_log(unique_data_type):
	"""
	The log is removed along with the ratings
	"""

	data_id = await post_logged(data_type=unique_data_type)

	await helpers.rate_object(nex.FRIEND_A, data_id, 5)

	async with helpers.connect(nex.FRIEND_A) as client:
		await client.reset_rating(helpers.rating_target(data_id, 0), constants.INVALID_PASSWORD)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

# * Which slots keep a log

@pytest.mark.parametrize("slot_fields", [
	pytest.param({}, id="no-flags-and-no-lock"),
	pytest.param({"flag": constants.RatingFlag.RATING_FLAG_ROUND_MINUS}, id="round-minus"),
	pytest.param({"flag": constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING}, id="disable-self-rating"),
	pytest.param({"flag": 0x40}, id="a-flag-which-is-not-defined")
])
async def test_a_slot_which_keeps_no_log(unique_data_type, slot_fields):
	"""
	Only some slots keep a log, and asking for one from any other slot is
	refused, whether or not it has been rated
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(**slot_fields)])

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id) == "DataStore::OperationNotAllowed"

@pytest.mark.parametrize("slot_fields", [
	pytest.param({"flag": constants.RatingFlag.RATING_FLAG_MODIFIABLE}, id="modifiable"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_INTERVAL, "period_duration": 0}, id="a-lock-which-never-locks"),
	pytest.param({"lock_type": constants.RatingLockType.RATING_LOCK_PERMANENT}, id="a-lock"),
	pytest.param({"flag": 0x01}, id="flag-0x01"),
	pytest.param({"flag": 0x02}, id="flag-0x02")
])
async def test_a_slot_which_keeps_a_log(unique_data_type, slot_fields):
	"""
	RATING_FLAG_MODIFIABLE, any lock type besides RATING_LOCK_NONE, and
	the unknown flag bits 0x01 and 0x02, all keep a log
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(**slot_fields)])

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

# * The lock expiration time

async def test_an_interval_lock_expires_after_its_duration(unique_data_type):
	"""
	period_duration seconds after the rating
	"""

	now, expiry = await rate_and_log({
		"lock_type": constants.RatingLockType.RATING_LOCK_INTERVAL,
		"period_duration": 3600
	}, unique_data_type)

	assert datetime.timedelta(hours=1) <= expiry - now <= datetime.timedelta(hours=1, seconds=10)

async def test_a_permanent_lock_never_expires(unique_data_type):
	"""
	The lock lasts until PERMANENT_DATE_TIME
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_PERMANENT)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 2)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == (True, nex.FRIEND_A, 2, constants.PERMANENT_DATE_TIME)

@pytest.mark.parametrize("period_duration, period_hour", [
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, 0, id="monday-at-midnight"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, 5, id="monday-at-5"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_MON, -5, id="monday-at-minus-5"),
	pytest.param(constants.RatingLockPeriod.RATING_LOCK_PERIOD_SUN, 23, id="sunday-at-23")
])
async def test_a_period_lock_on_a_day_of_the_week(unique_data_type, period_duration, period_hour):
	"""
	The lock lifts the next time it is that day at that hour. A negative
	hour counts back from midnight of that day, into the day before
	"""

	now, expiry = await rate_and_log({
		"lock_type": constants.RatingLockType.RATING_LOCK_PERIOD,
		"period_duration": period_duration,
		"period_hour": period_hour
	}, unique_data_type)

	assert expiry == next_weekday_at(now, period_duration, period_hour)

@pytest.mark.parametrize("period_duration, period_hour", [
	pytest.param(0, 0, id="the-next-midnight"),
	pytest.param(0, 23, id="the-next-23"),
	pytest.param(1, 5, id="a-day-after-the-next-5"),
	pytest.param(365, 0, id="365-days-after-the-next-midnight")
])
async def test_a_period_lock_of_a_number_of_days(unique_data_type, period_duration, period_hour):
	"""
	A positive period_duration is a number of days, counted from the next
	time it is the given hour
	"""

	now, expiry = await rate_and_log({
		"lock_type": constants.RatingLockType.RATING_LOCK_PERIOD,
		"period_duration": period_duration,
		"period_hour": period_hour
	}, unique_data_type)

	assert expiry == days_after_the_next_hour(now, period_duration, period_hour)

async def test_the_log_is_kept_after_an_interval_lock_lifts(unique_data_type):
	"""
	The user can rate again once the lock lifts, but the log still says
	they have rated, with the expiration time which has now passed
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(lock_type=constants.RatingLockType.RATING_LOCK_INTERVAL, period_duration=1)])

	await helpers.rate_object(nex.FRIEND_A, data_id, 6)
	before = log_fields(await get_rating_with_log(nex.FRIEND_A, data_id))

	await anyio.sleep(3)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == before
	assert before[:3] == (True, nex.FRIEND_A, 6)
	assert await helpers.rate_object_result(nex.FRIEND_A, data_id) is None

# * target.data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to read
	"""

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await get_rating_with_log_error(nex.FRIEND_A, 999999) == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects have no ratings to read
	"""

	data_id = await post_logged(data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE)])

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await post_logged(data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

# * target.slot

async def test_the_last_slot(unique_data_type):
	"""
	Slot RATING_SLOT_MAX is a slot like any other
	"""

	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[helpers.rating_slot(constants.RATING_SLOT_MAX, flag=constants.RatingFlag.RATING_FLAG_MODIFIABLE)])

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id, slot=constants.RATING_SLOT_MAX)) == unrated(nex.FRIEND_A)

async def test_a_slot_which_was_never_initialised(unique_data_type):
	"""
	Only the slots the object was posted with can be read
	"""

	data_id = await post_logged(data_type=unique_data_type)

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id, slot=1) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_RATING_SLOT, id="past-the-last"),
	pytest.param(-1, id="negative")
])
async def test_a_slot_which_does_not_exist(unique_data_type, slot):
	"""
	Only slots 0 to RATING_SLOT_MAX exist
	"""

	data_id = await post_logged(data_type=unique_data_type)

	assert await get_rating_with_log_error(nex.FRIEND_A, data_id, slot=slot) == "DataStore::InvalidArgument"

# * access_password

async def test_no_access_password_without_permission(unique_data_type):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post_logged(data_type=unique_data_type, permission=permission)

	assert await get_rating_with_log_error(nex.STRANGER, data_id) == "DataStore::PermissionDenied"

async def test_wrong_access_password_without_permission(unique_data_type):
	"""
	Using the wrong password on an object the caller has no access
	to fails with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post_logged(data_type=unique_data_type, permission=permission)

	assert await get_rating_with_log_error(nex.STRANGER, data_id, access_password=1234) == "DataStore::InvalidPassword"

async def test_wrong_access_password_with_permission(unique_data_type):
	"""
	Using the wrong password on an object the user already has access
	to is not an error
	"""

	data_id = await post_logged(data_type=unique_data_type)

	assert log_fields(await get_rating_with_log(nex.STRANGER, data_id, access_password=1234)) == unrated(nex.STRANGER)

async def test_access_password_bypasses_permissions(unique_data_type):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post_logged(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert log_fields(await get_rating_with_log(nex.STRANGER, data_id, access_password=password_info.access_password)) == unrated(nex.STRANGER)

async def test_the_update_password_is_not_the_access_password(unique_data_type):
	"""
	Reading only needs access, and the update password does not give it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post_logged(data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await get_rating_with_log_error(nex.STRANGER, data_id, access_password=password_info.update_password) == "DataStore::InvalidPassword"

# * Access permission tests
# *
# * Objects can always be read by their owner. For everyone else, it
# * depends on the access permission of the object

@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, permission, recipients, viewer, visible):
	"""
	Only users who can see an object can read its ratings and their log
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post_logged(data_type=unique_data_type, permission=access)

	if not visible:
		assert await get_rating_with_log_error(viewer, data_id) == "DataStore::PermissionDenied"
		return

	assert log_fields(await get_rating_with_log(viewer, data_id)) == unrated(viewer)

# * Object status tests
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

async def test_a_pending_object_can_be_read_by_its_owner(unique_data_type):
	"""
	The ratings of an object under review can still be read by its owner
	"""

	data_id = await post_logged(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await post_logged(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await get_rating_with_log_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_a_rejected_object_can_be_read_by_its_owner(unique_data_type):
	"""
	The ratings of a rejected object can still be read by its owner
	"""

	data_id = await helpers.post_rejected(post_logged, nex.FRIEND_A, data_type=unique_data_type)

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type):
	"""
	Rejected objects appear as under review still to other users
	"""

	data_id = await helpers.post_rejected(post_logged, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_rating_with_log_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

async def test_read_locks_do_not_stop_reading(unique_data_type):
	"""
	Read locks only gate downloading an object
	"""

	data_id = await post_logged(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_B, data_id=data_id, lock_id=1234)

	assert log_fields(await get_rating_with_log(nex.STRANGER, data_id)) == unrated(nex.STRANGER)

# * The rating slots settings

async def test_disable_self_rating_does_not_stop_the_owner_reading(unique_data_type):
	"""
	The owner may not rate the slot, but can still read it and their log
	"""

	slot = helpers.rating_slot(flag=constants.RatingFlag.RATING_FLAG_DISABLE_SELF_RATING | constants.RatingFlag.RATING_FLAG_MODIFIABLE)
	data_id = await helpers.post_rated(data_type=unique_data_type, slots=[slot])

	assert log_fields(await get_rating_with_log(nex.FRIEND_A, data_id)) == unrated(nex.FRIEND_A)

# * Reading the ratings does not count as referring to the object

async def test_reading_the_ratings_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move
	"""

	data_id = await post_logged(data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await get_rating_with_log(nex.STRANGER, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.expire_time.value() == before.expire_time.value()
