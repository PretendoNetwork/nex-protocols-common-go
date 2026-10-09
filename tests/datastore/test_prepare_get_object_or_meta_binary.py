import anyio
import pytest
from nintendo.nex import common, datastore, rmc

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def prepare_get(pid: int, **fields) -> rmc.RMCResponse:
	"""
	Asks for an object, returning both the download info and the
	additional metadata
	"""

	async with helpers.connect(pid) as client:
		return await client.prepare_get_object_or_meta_binary(helpers.prepare_get_param(**fields))

async def prepare_get_error(pid: int, **fields) -> str:
	"""
	Asks for an object and expects an error to be thrown, returning the
	name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.prepare_get_object_or_meta_binary(helpers.prepare_get_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PrepareGetObjectOrMetaBinary did not throw an error")

async def update_object(pid: int, data_id: int, size: int) -> int:
	"""
	Uploads a new version of an object on the file server, returning
	the version it was given
	"""

	async with helpers.connect(pid) as client:
		prepare_param = datastore.DataStorePrepareUpdateParam()
		prepare_param.data_id = data_id
		prepare_param.size = size
		prepare_param.update_password = constants.INVALID_PASSWORD
		prepare_param.extra_data = []

		update_info = await client.prepare_update_object(prepare_param)

		complete_param = datastore.DataStoreCompleteUpdateParam()
		complete_param.data_id = data_id
		complete_param.version = update_info.version
		complete_param.success = True

		await client.complete_update_object(complete_param)

		return update_info.version

# * What the server hands back
# *
# * Objects on the file server come back with somewhere to download them
# * from, and objects which are not come back with their meta binary
# * instead. The additional metadata is filled in either way, except for
# * the meta binary which is only sent when it is the objects contents

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_an_object_on_the_file_server(unique_data_type, post):
	"""
	There is somewhere to download the object from, and the meta binary
	is left out even though the object has one
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=1234, meta_binary=b"meta")
	response = await prepare_get(nex.FRIEND_A, data_id=data_id)

	# TODO - The stock servers also send some headers and such here, but since those are implementation details we don't check for them
	assert response.get_info.url != ""
	assert response.get_info.size == 1234

	assert response.additional_meta.owner_id == nex.FRIEND_A
	assert response.additional_meta.data_type == unique_data_type
	assert response.additional_meta.version == 1
	assert response.additional_meta.meta_binary == b""

async def test_an_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	There is nothing to download, so the object comes back as its meta
	binary instead
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"\x00\xFFcontents")
	response = await prepare_get(nex.FRIEND_A, data_id=data_id)

	assert response.get_info.url == ""
	assert response.get_info.size == 0
	assert response.get_info.headers == []
	assert response.get_info.root_ca_cert == b""
	assert response.get_info.data_id in [None, constants.INVALID_DATAID] # * Field is None when using NEX 3.4 and below

	assert response.additional_meta.owner_id == nex.FRIEND_A
	assert response.additional_meta.data_type == unique_data_type
	assert response.additional_meta.version == 1
	assert response.additional_meta.meta_binary == b"\x00\xFFcontents"

async def test_the_version_after_an_update(unique_data_type):
	"""
	The version is the latest one, and the size is the size of that
	version rather than the one the object was posted with
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=1234)
	version = await update_object(nex.FRIEND_A, data_id, 4321)
	response = await prepare_get(nex.FRIEND_A, data_id=data_id)

	assert version == 2
	assert response.additional_meta.version == version
	assert response.get_info.size == 4321

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

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects cannot be fetched
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * lock_id
# *
# * Read locks apply whether or not the object is on the file server

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_lock_id_which_is_not_the_one_holding_the_lock(unique_data_type, post):
	"""
	While a lock ID holds the lock, a different lock ID cannot get the
	object, even from the user who took the lock
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)

	assert await prepare_get_error(nex.FRIEND_A, data_id=data_id, lock_id=5678) == "DataStore::AlreadyLocked"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_lock_id_of_zero_releases_the_lock(unique_data_type, post):
	"""
	Asking for the object with no lock ID lets it go, so another lock
	ID can take it afterwards
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1234)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=0)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=5678)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_lock_id_on_an_object_which_has_no_read_lock(unique_data_type, post):
	"""
	An object which never asked for a read lock ignores the lock ID
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=999)
	await prepare_get(nex.FRIEND_A, data_id=data_id, lock_id=1)

# * persistence_target
# *
# * An object can be asked for by the slot it sits in rather than by its
# * data ID, and the slot is resolved the same way PrepareGetObject does

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_asking_by_persistence_target(unique_data_type, post):
	"""
	An object in a slot can be fetched through the slot, including by
	somebody other than its owner
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, helpers.PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=helpers.PERSISTENCE_SLOT)

	assert (await prepare_get(nex.FRIEND_A, persistence_target=target)).additional_meta.data_type == unique_data_type
	assert (await prepare_get(nex.STRANGER, persistence_target=target)).additional_meta.data_type == unique_data_type

async def test_the_target_is_used_over_the_data_id(unique_data_type):
	"""
	When both are set the slot wins and the data ID is ignored
	"""

	await helpers.post_persisted(helpers.post_meta_binary, nex.FRIEND_A, helpers.PERSISTENCE_SLOT, data_type=unique_data_type, meta_binary=b"in the slot")
	other = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"not in the slot")

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=helpers.PERSISTENCE_SLOT)
	response = await prepare_get(nex.FRIEND_A, data_id=other, persistence_target=target)

	assert response.additional_meta.meta_binary == b"in the slot"

async def test_a_target_with_no_slot_falls_back_on_the_data_id(unique_data_type):
	"""
	A target which doesn't use a slot is ignored and the data ID is used
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"by data id")
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.INVALID_PERSISTENCE_SLOT_ID)

	response = await prepare_get(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert response.additional_meta.meta_binary == b"by data id"

async def test_a_target_with_an_owner_of_zero(unique_data_type):
	"""
	An owner of 0 is treated as a real user, but obviously has no
	objects, and does not fall back to the caller
	"""

	await helpers.post_persisted(helpers.post_meta_binary, nex.FRIEND_A, helpers.PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=0, persistence_id=helpers.PERSISTENCE_SLOT)

	assert await prepare_get_error(nex.FRIEND_A, persistence_target=target) == "DataStore::NotFound"

async def test_a_target_naming_an_empty_slot(unique_data_type):
	"""
	An empty slot is missing rather than falling back on the data ID
	"""

	other = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=helpers.EMPTY_PERSISTENCE_SLOT)

	assert await prepare_get_error(nex.FRIEND_A, data_id=other, persistence_target=target) == "DataStore::NotFound"

async def test_a_target_naming_a_slot_which_cannot_exist():
	"""
	A slot out of range is simply empty, rather than a bad request
	"""

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.NUM_PERSISTENCE_SLOT)

	assert await prepare_get_error(nex.FRIEND_A, persistence_target=target) == "DataStore::NotFound"

# * access_password
# *
# * The password stands in for the access permission

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_right_access_password_gets_in(unique_data_type, post):
	"""
	Somebody who could not otherwise read the object can use its
	access password instead
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	response = await prepare_get(nex.STRANGER, data_id=data_id, access_password=password_info.access_password)

	assert response.additional_meta.owner_id == nex.FRIEND_A

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_wrong_access_password(unique_data_type, post):
	"""
	Sending a password which is not the right one says so
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_update_password_is_not_the_access_password(unique_data_type, post):
	"""
	Passwords only work in the place they belong to
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await prepare_get_error(nex.STRANGER, data_id=data_id, access_password=password_info.update_password) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_wrong_access_password_from_the_owner(unique_data_type, post):
	"""
	The password is only checked when the caller would be denied
	without one, so the owner can send anything
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id, access_password=1234)).additional_meta.owner_id == nex.FRIEND_A

# * extra_data
# TODO - The only NN NEX server available at the time of writing is Pokemon Bank, which doesn't support this field. These tests are written assuming the current implementations work

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param([], id="none-at-all"),
	pytest.param(["WUP", "4", "EUR", "110", "GB", ""], id="what-a-console-sends")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_extra_data_is_not_in_the_way(unique_data_type, extra_data, post):
	"""
	Just make sure that extraData doesn't kill the request
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id, extra_data=extra_data)).additional_meta.owner_id == nex.FRIEND_A

# * The access permission of the object

@pytest.mark.parametrize("permission, recipients, allowed, denied", helpers.ACCESS_PERMISSION_VIEWERS_AND_OUTSIDERS)
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_access_permission_of_the_object(unique_data_type, permission, recipients, allowed, denied, post):
	"""
	Fetching is gated the same way reading the metadata is
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=access)

	assert (await prepare_get(allowed, data_id=data_id)).additional_meta.owner_id == nex.FRIEND_A

	if denied is not None:
		assert await prepare_get_error(denied, data_id=data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("caller", [
	pytest.param(nex.FRIEND_B, id="a-friend"),
	pytest.param(nex.STRANGER, id="a-stranger")
])
@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_private_object_is_closed_to_everybody(unique_data_type, caller, post):
	"""
	A private object lets nobody in at all
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).additional_meta.owner_id == nex.FRIEND_A
	assert await prepare_get_error(caller, data_id=data_id) == "DataStore::PermissionDenied"

# * The object status

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_pending_object(unique_data_type, post):
	"""
	An object under review can still be fetched by its owner, but
	everybody else is told it is under review
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).additional_meta.owner_id == nex.FRIEND_A
	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_rejected_object(unique_data_type, post):
	"""
	A rejected object can still be fetched by its owner, and being
	rejected is not told apart from being under review for everybody else
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert (await prepare_get(nex.FRIEND_A, data_id=data_id)).additional_meta.owner_id == nex.FRIEND_A
	assert await prepare_get_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

# * Asking for an object counts as referring to it, whether or not it is
# * on the file server

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_getting_an_object_refers_to_it(unique_data_type, post):
	"""
	The reference count goes up and the referred time moves, without
	the caller having to touch the object separately
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count + 1
	assert after.referred_time.value() > before.referred_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.POST_METHODS)
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
@pytest.mark.parametrize("post", helpers.POST_METHODS)
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

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_getting_an_object_does_not_move_the_expiry(unique_data_type, post):
	"""
	An object without DATA_FLAG_PERIOD_FROM_LAST_REFERRED set does not
	have its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() == before.expire_time.value()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_getting_an_object_can_push_the_expiry_back(unique_data_type, post):
	"""
	An object with DATA_FLAG_PERIOD_FROM_LAST_REFERRED set has its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await prepare_get(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() > before.expire_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.POST_METHODS)
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

@pytest.mark.parametrize("post", helpers.POST_METHODS)
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
