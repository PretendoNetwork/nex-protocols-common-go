import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT = 3

async def delete_error(pid: int, **fields) -> str:
	"""
	Deletes an object and expects an error to be thrown, returning
	the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.delete_object(helpers.delete_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::DeleteObject did not throw an error")

# * What deleting does

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deletes_an_object(unique_data_type, post):
	"""
	Deleted objects behave as if they never existed
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deleting_twice(unique_data_type, post):
	"""
	An object can only be deleted once
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await delete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_deleting_a_persisted_object_empties_the_slot(unique_data_type, post):
	"""
	Deleting an object which is being persisted leaves its
	persistence slot empty
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=SLOT)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, persistence_init_param=persistence_init_param)

	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT) == "DataStore::NotFound"

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UINT64_MAX, id="uint64-max")
])
async def test_unknown_data_id(data_id):
	"""
	Deleting an object which does not exist fails
	"""

	assert await delete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("data_id", [
	pytest.param(1, id="the-very-bottom"),
	pytest.param(500000, id="half-way-up"),
	pytest.param(900000, id="the-start-of-the-system-range"),
	pytest.param(999999, id="the-last-system-data-id")
])
async def test_system_range_data_ids(data_id):
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await delete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_deleting_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet,
	so they cannot be deleted
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await delete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * update_password
# *
# * Objects may carry an update password, which acts in place of the
# * delete permission

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_update_password_replaces_the_permission(unique_data_type, post):
	"""
	The correct update password lets anybody delete the object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	await helpers.delete_object(nex.STRANGER, data_id=data_id, update_password=password_info.update_password)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_update_password(unique_data_type, post):
	"""
	Sending the wrong password fails
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await delete_error(nex.STRANGER, data_id=data_id, update_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_access_password_is_not_the_update_password(unique_data_type, post):
	"""
	Passwords only work for what they were made for, so the access
	password of an object is no use for deleting it
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await delete_error(nex.STRANGER, data_id=data_id, update_password=password_info.access_password) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_update_password_as_the_owner(unique_data_type, post):
	"""
	The password is only checked when the caller would be denied
	without one, so the owner can send anything
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id, update_password=1234)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * Deleting an object is gated by the objects delete permission, which
# * is separate from the access permission used to read it

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_delete_permission_private(unique_data_type, post):
	"""
	Only the owner may delete the object, which is the default
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await delete_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"
	assert await delete_error(nex.FRIEND_B, data_id=data_id) == "DataStore::PermissionDenied"

	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_delete_permission_public(unique_data_type, post):
	"""
	Anybody may delete the object
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=permission)

	await helpers.delete_object(nex.STRANGER, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_delete_permission_friend(unique_data_type, post):
	"""
	Only friends of the owner may delete the object
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_FRIEND)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=permission)

	assert await delete_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

	await helpers.delete_object(nex.FRIEND_B, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_delete_permission_specified(unique_data_type, post):
	"""
	Only the listed users may delete the object, whether or not they
	are friends of the owner
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=permission)

	assert await delete_error(nex.FRIEND_B, data_id=data_id) == "DataStore::PermissionDenied"

	await helpers.delete_object(nex.STRANGER, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_delete_permission_specified_friend(unique_data_type, post):
	"""
	Only listed users who are also friends of the owner may delete
	the object
	"""

	to_the_friend = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND, recipients=[nex.FRIEND_B])
	to_the_stranger = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED_FRIEND, recipients=[nex.STRANGER])

	for_the_friend = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=to_the_friend)
	for_the_stranger = await post(nex.FRIEND_A, data_type=unique_data_type, delete_permission=to_the_stranger)

	assert await delete_error(nex.STRANGER, data_id=for_the_stranger) == "DataStore::PermissionDenied"
	assert await delete_error(nex.FRIEND_B, data_id=for_the_stranger) == "DataStore::PermissionDenied"

	await helpers.delete_object(nex.FRIEND_B, data_id=for_the_friend)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_access_permission_is_not_used(unique_data_type, post):
	"""
	The two permissions are independent, so a user who cannot even
	read the object may still be allowed to delete it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	deletable = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission, delete_permission=deletable)

	assert await helpers.get_meta_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

	await helpers.delete_object(nex.STRANGER, data_id=data_id)

# * Objects which are in an unusual state

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deleting_an_object_under_review(unique_data_type, post):
	"""
	Objects which are under review are hidden from everybody, but
	their owner can still delete them
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_deleting_a_read_locked_object(unique_data_type, post):
	"""
	Read locks only gate downloads of an object, so one which is
	locked by somebody else can still be deleted
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	await helpers.touch_object(nex.STRANGER, data_id=data_id, lock_id=1234)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)
