import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Timestamps are only accurate to the second, so the tests which
# * check them have to wait before touching
TIMESTAMP_DELAY = 2

async def touch_error(pid: int, **fields) -> str:
	"""
	Touches an object and expects an error to be thrown, returning
	the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.touch_object(helpers.touch_object_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::TouchObject did not throw an error")

# * What touching an object is for
# *
# * Touching an object references it, without downloading it or
# * returning any metadata

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_increments_the_reference_count(unique_data_type, post):
	"""
	Every touch counts as one reference
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == 0

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == 1

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == 2

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_reference_count_stops_at_its_limit(unique_data_type, post):
	"""
	A count already at UINT32_MAX stays there rather than wrapping
	back around to 0
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=helpers.UINT32_MAX)
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == helpers.UINT32_MAX

@pytest.mark.parametrize("touches, referred_count", [
	pytest.param(0, 10, id="raised"),
	pytest.param(3, 1, id="lowered")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_references_carry_on_from_a_changed_referred_count(unique_data_type, touches, referred_count, post):
	"""
	The next reference counts up from the value which was set, not from
	what the count was before
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	for _ in range(touches):
		await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=referred_count)
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == referred_count + 1

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_failed_touch_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Being refused the object does not count as having referred to it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await touch_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_touch_blocked_by_the_read_lock_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Being turned away by a lock ID which does not hold the read lock
	does not count as having referred to the object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await touch_error(nex.STRANGER, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_updates_the_reference_time_and_validity_period(unique_data_type, post):
	"""
	Touching always updates the time the object was last referenced, but
	only extends the validity period when the object was posted with
	DATA_FLAG_PERIOD_FROM_LAST_REFERRED
	"""

	plain = await post(nex.FRIEND_A, data_type=unique_data_type)
	extends = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)

	before = {
		plain: await helpers.get_meta(nex.FRIEND_A, data_id=plain),
		extends: await helpers.get_meta(nex.FRIEND_A, data_id=extends)
	}

	await anyio.sleep(TIMESTAMP_DELAY)

	await helpers.touch_object(nex.FRIEND_A, data_id=plain)
	await helpers.touch_object(nex.FRIEND_A, data_id=extends)

	after = {
		plain: await helpers.get_meta(nex.FRIEND_A, data_id=plain),
		extends: await helpers.get_meta(nex.FRIEND_A, data_id=extends)
	}

	assert after[plain].referred_time.value() > before[plain].referred_time.value()
	assert after[extends].referred_time.value() > before[extends].referred_time.value()

	assert after[plain].expire_time.value() == before[plain].expire_time.value()
	assert after[extends].expire_time.value() > before[extends].expire_time.value()

# * data_id
# *
# * Objects which cannot be touched

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid")
])
async def test_unknown_data_id(data_id):
	"""
	Touching an object which does not exist fails
	"""

	assert await touch_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_touching_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects cannot be touched
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await touch_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_touching_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await touch_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * lock_id
# *
# * Objects posted with DATA_FLAG_USE_READ_LOCK remember the lock ID they
# * were last referenced with, and only that lock ID may be used until
# * someone releases the lock with a lock ID of 0

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_lock_id_locks_the_object(unique_data_type, post):
	"""
	The lock ID the object is referenced with is the only one
	allowed from then on until its released
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234) # * Initial lock set
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234) # * Verifying that the object can be accessed with the lock ID

	assert await touch_error(nex.FRIEND_A, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_owner_is_not_exempt_from_the_lock(unique_data_type, post):
	"""
	Locking an object locks it for everyone, including its owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.STRANGER, data_id=data_id, lock_id=1234)

	assert await touch_error(nex.FRIEND_A, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_lock_id_of_zero_releases_the_lock(unique_data_type, post):
	"""
	A lock ID of 0 releases the lock
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=0)
	await helpers.touch_object(nex.STRANGER, data_id=data_id, lock_id=5678)

	assert await touch_error(nex.FRIEND_A, data_id=data_id, lock_id=1234) == "DataStore::AlreadyLocked"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_failed_lock_leaves_the_lock_alone(unique_data_type, post):
	"""
	Using the wrong lock ID does not change the lock
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)

	assert await touch_error(nex.FRIEND_A, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_lock_ids_are_ignored_without_the_flag(unique_data_type, post):
	"""
	Objects which were not posted with DATA_FLAG_USE_READ_LOCK are
	never locked, so any lock ID can be used on them
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=5678)

# * access_password

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_access_password_bypasses_permissions(unique_data_type, post):
	"""
	The correct password lets anyone touch the object
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	await helpers.touch_object(nex.STRANGER, data_id=data_id, access_password=password_info.access_password)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_access_password_without_permission(unique_data_type, post):
	"""
	Using the wrong password on an object the caller has no access to fails
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await touch_error(nex.STRANGER, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_access_password_as_the_owner(unique_data_type, post):
	"""
	The password is only checked when the caller would be denied
	access without one, so the owner can send anything
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, access_password=1234)

# * Access permissions

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_anyone_with_access_can_touch(unique_data_type, post):
	"""
	Touching is a read operation, so it is not limited to the owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await helpers.touch_object(nex.STRANGER, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == 1

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_touching_an_object_without_access(unique_data_type, post):
	"""
	Objects the caller cannot access cannot be touched
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await touch_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_friend_of_the_owner_can_touch(unique_data_type, post):
	"""
	PERMISSION_FRIEND objects can be referenced by the friends
	of the object owner
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_FRIEND)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	await helpers.touch_object(nex.FRIEND_B, data_id=data_id)

	assert await touch_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_specified_friend_needs_both(unique_data_type, post):
	"""
	PERMISSION_SPECIFIED_FRIEND objects can only be referenced by
	recipients who are also friends of the owner
	"""

	to_the_friend = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND, recipients=[nex.FRIEND_B])
	to_the_stranger = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND, recipients=[nex.STRANGER])

	for_the_friend = await post(nex.FRIEND_A, data_type=unique_data_type, permission=to_the_friend)
	for_the_stranger = await post(nex.FRIEND_A, data_type=unique_data_type, permission=to_the_stranger)

	# * A recipient who is also a friend
	await helpers.touch_object(nex.FRIEND_B, data_id=for_the_friend)

	# * A recipient who is not a friend
	assert await touch_error(nex.STRANGER, data_id=for_the_stranger) == "DataStore::PermissionDenied"

	# * A friend who is not a recipient
	assert await touch_error(nex.FRIEND_B, data_id=for_the_stranger) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_specified_does_not_need_friendship(unique_data_type, post):
	"""
	PERMISSION_SPECIFIED objects can be referenced by their recipients,
	whether or not they are friends of the owner
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	await helpers.touch_object(nex.STRANGER, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_pending_object_can_be_touched_by_its_owner(unique_data_type, post):
	"""
	Objects under review can still be referenced by their owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)
