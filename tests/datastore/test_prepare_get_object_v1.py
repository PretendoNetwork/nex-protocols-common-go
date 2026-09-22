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
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, nex.STRANGER, id="specified-friend"),
]

async def prepare_get(pid: int, **fields) -> datastore.DataStoreReqGetInfoV1:
	"""
	Asks where to download an object from, returning all the info
	"""

	async with helpers.connect(pid) as client:
		return await client.prepare_get_object_v1(helpers.prepare_get_param_v1(**fields))

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
			await client.prepare_get_object_v1(helpers.prepare_get_param_v1(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PrepareGetObjectV1 did not throw an error")

async def post_rejected(post, pid: int, **fields) -> int:
	"""
	Posts an object and marks it as rejected, as the given user
	"""

	data_id = await post(pid, **fields)

	await helpers.change_meta(pid, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=constants.DataStatus.DATA_STATUS_REJECTED)

	return data_id

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
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(helpers.UINT32_MAX, id="uint32-max"),
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
	pytest.param(nex.STRANGER, id="a-stranger"),
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

	data_id = await post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Being rejected is not told apart from being under review
	"""

	data_id = await post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

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
