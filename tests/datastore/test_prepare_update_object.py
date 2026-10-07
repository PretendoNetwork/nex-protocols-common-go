import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

POSTED_SIZE = 10
UPDATED_SIZE = 20

# * Each update permission, who tries to update with it, and whether they may
UPDATE_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.STRANGER, False, id="private"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, True, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, True, id="friend-to-a-friend"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.STRANGER, False, id="friend-to-a-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, True, id="specified-to-a-recipient"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, True, id="specified-friend-to-a-friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], nex.STRANGER, False, id="specified-friend-to-a-stranger")
]

async def prepare(pid: int, **fields) -> datastore.DataStoreReqUpdateInfo:
	"""
	Starts updating an object, returning all the info
	"""

	fields.setdefault("size", UPDATED_SIZE)

	async with helpers.connect(pid) as client:
		return await client.prepare_update_object(helpers.prepare_update_param(**fields))

async def prepare_error(pid: int, **fields) -> str:
	"""
	Starts updating an object and expects an error to be thrown,
	returning the name of the error
	"""

	fields.setdefault("size", UPDATED_SIZE)

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.prepare_update_object(helpers.prepare_update_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PrepareUpdateObject did not throw an error")

# * What the server hands back

async def test_returns_somewhere_to_upload_to(unique_data_type):
	"""
	Basic functionality test
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert info.url != ""
	assert [field.key for field in info.form] != [] # TODO - Maybe this should be allowed, for cases where the file server doesn't need addtional form fields?

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_first_update_is_version_2(unique_data_type, post):
	"""
	Every object starts at version 1, and an update is for the version
	after the current one
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert info.version == 2

async def test_nothing_changes_until_the_update_is_completed(unique_data_type):
	"""
	Preparing only reserves the new version. The object keeps its old
	version, size, update time and expiry until the update is completed
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await prepare(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await helpers.version_of(data_id) == 1
	assert after.size == POSTED_SIZE
	assert after.update_time.value() == before.update_time.value()
	assert after.expire_time.value() == before.expire_time.value()

async def test_preparing_twice_hands_out_a_new_version_each_time(unique_data_type):
	"""
	An update which has not been completed does not stop another one
	from being prepared, which gets the version after it
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	first = await prepare(nex.FRIEND_A, data_id=data_id)
	second = await prepare(nex.FRIEND_A, data_id=data_id)

	assert [first.version, second.version] == [2, 3]

async def test_versions_carry_on_after_a_completed_update(unique_data_type):
	"""
	Once an update is completed, the next one is for the version after it
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	first = await prepare(nex.FRIEND_A, data_id=data_id)

	await helpers.complete_update_object(nex.FRIEND_A, data_id=data_id, version=first.version)
	second = await prepare(nex.FRIEND_A, data_id=data_id)

	assert second.version == 3

async def test_a_failed_upload_still_uses_up_its_version(unique_data_type):
	"""
	The version handed out for an update which failed is never handed
	out again
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	first = await prepare(nex.FRIEND_A, data_id=data_id)

	await helpers.complete_update_object(nex.FRIEND_A, data_id=data_id, version=first.version, success=False)
	second = await prepare(nex.FRIEND_A, data_id=data_id)

	assert second.version == 3

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is nothing to update
	"""

	assert await prepare_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await prepare_error(nex.FRIEND_A, data_id=999999) == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects cannot be updated
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await prepare_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)

	assert await prepare_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_an_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	There is no file to replace, so the object exists but cannot be
	updated this way
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await prepare_error(nex.FRIEND_A, data_id=data_id) == "DataStore::OperationNotAllowed"

# * size

@pytest.mark.parametrize("size", [
	pytest.param(0, id="empty"),
	pytest.param(helpers.MAX_OBJECT_SIZE + 1, id="over-the-upload-limit")
])
async def test_any_size_is_allowed(unique_data_type, size):
	"""
	Unlike posting, the game server does not limit the size of an update,
	and the new size is what the object has once the update is completed.
	Nothing is uploaded here, but a real upload over the posting limit
	was accepted by the file server on Pokemon Bank and downloaded intact
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id, size=size)

	await helpers.complete_update_object(nex.FRIEND_A, data_id=data_id, version=info.version)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == size

# * update_password
# *
# * Objects may carry an update password, which acts in place of the
# * update permission

async def test_update_password_replaces_the_permission(unique_data_type):
	"""
	Someone without update permission can still update the object
	with its update password
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)
	info = await prepare(nex.STRANGER, data_id=data_id, update_password=password_info.update_password)

	assert info.version == 2

async def test_wrong_update_password(unique_data_type):
	"""
	A password which does not match is refused
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)

	assert await prepare_error(nex.STRANGER, data_id=data_id, update_password=1234) == "DataStore::InvalidPassword"

async def test_the_access_password_is_not_the_update_password(unique_data_type):
	"""
	The access password only gives read access, not update access
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)

	assert await prepare_error(nex.STRANGER, data_id=data_id, update_password=password_info.access_password) == "DataStore::InvalidPassword"

async def test_wrong_update_password_as_the_owner(unique_data_type):
	"""
	The owner already has update permission, so the password is
	never checked
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id, update_password=1234)

	assert info.version == 2

# * extra_data
# *
# * Disclaimer: these tests were not ran against a live server, as at the time
# * of writing the only active NEX server on Nintendo Network is Pokemon Bank,
# * which is a NEX 3.4 server, and this field was added in NEX 3.5. The tests
# * are written to assume the current implementation is correct, and will need
# * to be updated at some point when more testing is done on NSO

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
async def test_no_extra_data_at_all(unique_data_type):
	"""
	A client older than NEX 3.5 does not send the field, which is not
	the same as sending something wrong
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id, extra_data=[])

	assert info.version == 2

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="extra_data was added in NEX 3.5")
@pytest.mark.parametrize("extra_data", [
	pytest.param(["CTR", "2", "EUR", "110", "GB", ""], id="a-european-3ds"),
	pytest.param(["WUP", "2", "USA", "49", "US", ""], id="an-american-wii-u")
])
async def test_extra_data_a_console_would_send(unique_data_type, extra_data):
	"""
	Both platforms are accepted
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id, extra_data=extra_data)

	assert info.version == 2

# * The objects update permission
# *
# * Updating an object is gated by the objects update permission, which
# * is separate from the access permission used to read it

@pytest.mark.parametrize("permission, recipients, pid, allowed", UPDATE_PERMISSIONS)
async def test_the_update_permission(unique_data_type, permission, recipients, pid, allowed):
	"""
	Only users the update permission allows can update the object. The
	owner always can, which every other test relies on
	"""

	update = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, delete_permission=update)

	if not allowed:
		assert await prepare_error(pid, data_id=data_id) == "DataStore::PermissionDenied"
		return

	info = await prepare(pid, data_id=data_id)

	assert info.version == 2

async def test_the_access_permission_is_not_used(unique_data_type):
	"""
	Someone who may update an object can do so even if they cannot read it
	"""

	access = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, permission=access, delete_permission=update)
	info = await prepare(nex.STRANGER, data_id=data_id)

	assert info.version == 2

# * The object status

async def test_an_object_under_review_can_be_updated_by_its_owner(unique_data_type):
	"""
	The owner can update their object while it is still under review
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert info.version == 2

async def test_an_object_under_review_is_under_review_for_everyone_else(unique_data_type):
	"""
	Everyone but the owner is told it is under review, whatever their
	update permission is
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await prepare_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner. Pokemon Bank now lets the owner keep using
#        them, so this matches Pokemon Bank until more information is confirmed
async def test_a_rejected_object_can_be_updated_by_its_owner(unique_data_type):
	"""
	Rejecting an object does not stop its owner from updating it
	"""

	data_id = await helpers.post_rejected(helpers.post_object, nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert info.version == 2

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired, so there is
	nothing to update
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await prepare_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_a_read_locked_object(unique_data_type):
	"""
	The read lock only guards downloads, so an object locked by someone
	else can still be updated
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	async with helpers.connect(nex.STRANGER) as client:
		await client.prepare_get_object(helpers.prepare_get_param(data_id=data_id, lock_id=1234))

	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert info.version == 2

# * Notifications

async def test_the_update_notification_waits_for_completion(unique_data_type):
	"""
	Recipients of an object with DATA_FLAG_USE_NOTIFICATION_ON_UPDATE are
	not notified when the update is prepared, only once it is completed
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE)
	info = await prepare(nex.FRIEND_A, data_id=data_id)

	assert await helpers.notified(data_id) == []

	await helpers.complete_update_object(nex.FRIEND_A, data_id=data_id, version=info.version)

	assert await helpers.notified(data_id) == [nex.STRANGER]

# * Preparing an update does not count as referring to the object

async def test_preparing_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await prepare(nex.FRIEND_A, data_id=data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
