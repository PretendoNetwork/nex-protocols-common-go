import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

# * Getting an object is gated by its access permission, so each one is
# * tried against somebody it lets in and somebody it keeps out
ACCESS_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, None, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, nex.STRANGER, id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, nex.FRIEND_B, id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, nex.STRANGER, id="specified-friend")
]

SLOT = 0
EMPTY_SLOT = 15

async def prepare_get(pid: int, **fields) -> datastore.DataStoreReqGetInfo:
	"""
	Asks where to download an object from, returning all the info
	"""

	async with helpers.connect(pid) as client:
		return await client.prepare_get_object(helpers.prepare_get_param(**fields))

async def prepare_get_error(pid: int, **fields) -> str:
	"""
	Asks where to download an object from and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.prepare_get_object(helpers.prepare_get_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PrepareGetObject did not throw an error")

# * What the server hands back

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_returns_somewhere_to_download_from(unique_data_type, post):
	"""
	Basic functionality test
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	info = await prepare_get(nex.FRIEND_A, data_id=data_id)

	# TODO - The stock servers also send some headers and such here, but since those are implementation details we don't check for them
	assert info.url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_size_is_the_one_the_object_was_posted_with(unique_data_type, post):
	"""
	The size comes from the object, not from whatever was uploaded
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=1234)
	info = await prepare_get(nex.FRIEND_A, data_id=data_id)

	assert info.size == 1234

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="zero"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	A data ID with no object behind it is missing, including INVALID_DATAID
	"""

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects have nothing to download
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_an_object_with_no_file_to_download(unique_data_type):
	"""
	An object posted with PostMetaBinary has no file on the file
	server, so there is nowhere to point the caller at
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id) == "DataStore::OperationNotAllowed"

# * lock_id

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_taking_the_read_lock(unique_data_type, post):
	"""
	The first caller to name a lock ID takes the lock, and can keep
	getting the object with it
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_lock_id_which_is_not_the_one_holding_the_lock(unique_data_type, post):
	"""
	While somebody holds the lock, a different lock ID cannot get the
	object, even if that caller is is the one who created the lock
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)

	assert await prepare_get_error(nex.FRIEND_B, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"
	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_lock_id_of_zero_releases_the_lock(unique_data_type, post):
	"""
	Asking for the object with no lock ID lets it go, so another lock
	ID can take it afterwards. The user releasing the lock does not need
	to be the one who created the lock
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await prepare_get(nex.FRIEND_B, data_id=data_id, lock_id=0) # * This user did not create the lock, but can release it anyway
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=5678)

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_lock_is_not_per_user(unique_data_type, post):
	"""
	The lock belongs to the lock ID rather than to whoever took it, so
	anybody naming it gets the object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await prepare_get(nex.STRANGER, data_id=data_id, lock_id=1234)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_lock_id_on_an_object_which_has_no_read_lock(unique_data_type, post):
	"""
	An object which never asked for a read lock ignores the lock ID
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=999)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1)

# * persistence_target
# *
# * An object can be asked for by the slot it sits in rather than by its data ID

async def test_asking_by_persistence_target(unique_data_type):
	"""
	An object in a slot can be fetched through the slot
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param, size=111)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=SLOT)
	info = await prepare_get(nex.FRIEND_A, persistence_target=target)

	assert info.size == 111

async def test_the_target_is_used_over_the_data_id(unique_data_type):
	"""
	When both are set the slot wins and the data ID is ignored
	"""

	# * Have to use different sizes as a hack to know which object came back
	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param, size=111)
	other = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=222)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=SLOT)
	info = await prepare_get(nex.FRIEND_A, data_id=other, persistence_target=target)

	assert info.size == 111

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_target_with_no_slot_falls_back_on_the_data_id(unique_data_type, post):
	"""
	A target which doesn't use a slot is ignored and the data ID is used
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=222)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.INVALID_PERSISTENCE_SLOT_ID)

	info = await prepare_get(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert info.size == 222

async def test_a_target_with_an_owner_of_zero(unique_data_type):
	"""
	An owner of 0 is treated as a real user, but obviously has no objects,
	and does not fall back to the data ID or the caller PID as the owner
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param)
	other = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)

	target = helpers.persistence_target(owner_id=0, persistence_id=SLOT)

	assert await prepare_get_error(nex.FRIEND_A, data_id=other, persistence_target=target) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_target_naming_an_empty_slot(unique_data_type, post):
	"""
	An empty slot is missing rather than falling back on the data ID
	"""

	other = await post(nex.FRIEND_A, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=EMPTY_SLOT)

	assert await prepare_get_error(nex.FRIEND_A, data_id=other, persistence_target=target) == "DataStore::NotFound"

async def test_a_target_naming_an_owner_who_has_nothing():
	"""
	A user who has never persisted anything has no slots to read
	"""

	target = helpers.persistence_target(owner_id=1234, persistence_id=SLOT)

	assert await prepare_get_error(nex.FRIEND_A, persistence_target=target) == "DataStore::NotFound"

async def test_a_target_naming_a_slot_which_cannot_exist():
	"""
	Unlike GetPersistenceInfo, which calls this a bad request, here a
	slot out of range is simply empty
	"""

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.NUM_PERSISTENCE_SLOT)

	assert await prepare_get_error(nex.FRIEND_A, persistence_target=target) == "DataStore::NotFound"

async def test_neither_a_data_id_nor_a_target():
	"""
	A request which names nothing at all finds nothing, rather than
	being refused as a bad request
	"""

	assert await prepare_get_error(nex.FRIEND_A) == "DataStore::NotFound"

@pytest.mark.parametrize("permission, recipients, allowed, denied", ACCESS_PERMISSIONS)
async def test_the_access_permission_of_an_object_in_a_slot(unique_data_type, permission, recipients, allowed, denied):
	"""
	Reaching an object through a slot is gated the same way reaching by data ID
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, permission=access, persistence_init_param=persistence_init_param)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=SLOT)

	assert (await prepare_get(allowed, persistence_target=target)).url != ""

	if denied is not None:
		assert await prepare_get_error(denied, persistence_target=target) == "DataStore::PermissionDenied"

# * access_password
# *
# * The password stands in for the access permission

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_right_access_password_gets_in(unique_data_type, post):
	"""
	Somebody who could not otherwise read the object can use its
	access password instead
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	info = await prepare_get(nex.STRANGER, data_id=data_id, access_password=password_info.access_password)

	assert info.url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_wrong_access_password(unique_data_type, post):
	"""
	Sending a password which is not the right one says so
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_update_password_is_not_the_access_password(unique_data_type, post):
	"""
	Passwords only work in the place they belong to
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, access_password=password_info.update_password) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_no_access_password_on_an_object_which_needs_one(unique_data_type, post):
	"""
	Without a password the caller is turned away on the permission
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_wrong_access_password_from_the_owner(unique_data_type, post):
	"""
	The owner does not need the password, for likely the same reason as
	test_a_wrong_access_password_from_somebody_who_has_access
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id, access_password=1234)).url != ""

@pytest.mark.parametrize("permission, recipients, allowed, denied", ACCESS_PERMISSIONS)
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_wrong_access_password_from_somebody_who_has_access(unique_data_type, permission, recipients, allowed, denied, post):
	"""
	The permission is looked at before the password, so somebody the
	object already lets in is not turned away by a wrong password
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=access)

	assert (await prepare_get(allowed, data_id=data_id, access_password=1234)).url != ""

	if denied is not None:
		assert await prepare_get_error(denied, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

# * extra_data
# TODO - The only NN NEX server available at the time of writing is Pokemon Bank, which doesn't support this field. These tests are written assuming the current implementations work

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param([], id="none-at-all"),
	pytest.param(["WUP", "4", "EUR", "110", "GB", ""], id="what-a-console-sends")
])
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_extra_data_is_not_in_the_way(unique_data_type, extra_data, post):
	"""
	Just make sure that extraData doesn't kill the request
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id, extra_data=extra_data)).url != ""

# * The access permission of the object

@pytest.mark.parametrize("permission, recipients, allowed, denied", ACCESS_PERMISSIONS)
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_access_permission_of_the_object(unique_data_type, permission, recipients, allowed, denied, post):
	"""
	Downloading is gated the same way reading the metadata is
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=access)

	assert (await prepare_get(allowed, data_id=data_id)).url != ""

	if denied is not None:
		assert await prepare_get_error(denied, data_id=data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("caller", [
	pytest.param(nex.FRIEND_B, id="a-friend"),
	pytest.param(nex.STRANGER, id="a-stranger")
])
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_private_object_is_closed_to_everybody(unique_data_type, caller, post):
	"""
	A private object lets nobody in at all
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).url != ""
	assert await prepare_get_error(caller, data_id=data_id) == "DataStore::PermissionDenied"

# * The object status

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_pending_object_is_downloadable_by_its_owner(unique_data_type, post):
	"""
	An object under review can still be fetched by its owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_rejected_object_is_downloadable_by_its_owner(unique_data_type, post):
	"""
	A rejected object can still be fetched by its owner
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Being rejected is not told apart from being under review
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

# * Asking where to download an object counts as referring to it

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_refers_to_it(unique_data_type, post):
	"""
	The reference count goes up and the referred time moves, without
	the caller having to touch the object separately
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count + 1
	assert after.referred_time.value() > before.referred_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_reference_count_stops_at_its_limit(unique_data_type, post):
	"""
	A count already at UINT32_MAX stays there rather than wrapping
	back around to 0
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=helpers.UINT32_MAX)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == helpers.UINT32_MAX

@pytest.mark.parametrize("gets, referred_count", [
	pytest.param(0, 10, id="raised"),
	pytest.param(3, 1, id="lowered")
])
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_references_carry_on_from_a_changed_referred_count(unique_data_type, gets, referred_count, post):
	"""
	The next reference counts up from the value which was set, not from
	what the count was before
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	for _ in range(gets):
		await prepare_get(nex.FRIEND_A, data_id=data_id)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=referred_count)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == referred_count + 1

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_does_not_move_the_expiry(unique_data_type, post):
	"""
	An object without DATA_FLAG_PERIOD_FROM_LAST_REFERRED set does not
	have its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() == before.expire_time.value()

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_can_push_the_expiry_back(unique_data_type, post):
	"""
	An object with DATA_FLAG_PERIOD_FROM_LAST_REFERRED set has its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() > before.expire_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_failed_get_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Being refused the object does not count as having referred to it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_get_blocked_by_the_read_lock_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Being turned away by a lock ID which does not hold the read lock
	does not count as having referred to the object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
