import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def complete(pid: int, **fields) -> None:
	"""
	Completes an update as the given user
	"""

	async with helpers.connect(pid) as client:
		await client.complete_update_object(helpers.complete_update_param(**fields))

async def complete_error(pid: int, **fields) -> str:
	"""
	Completes an update and expects an error to be thrown, returning
	the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.complete_update_object(helpers.complete_update_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::CompleteUpdateObject did not throw an error")

async def prepare(pid: int, data_id: int, size: int = helpers.UPDATED_SIZE, **fields) -> int:
	"""
	Prepares an update of the object as the given user, returning the
	version handed out
	"""

	return (await helpers.prepare_update_object(pid, data_id=data_id, size=size, **fields)).version

# * What completing an update does

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_applies_the_update(unique_data_type, post):
	"""
	The object moves to the version which was handed out, with the size
	the update was prepared with
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert await helpers.version_of(data_id) == 2
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == helpers.UPDATED_SIZE

async def test_the_update_time_and_expiry_move_when_the_update_is_completed(unique_data_type):
	"""
	The object counts as updated when the update is completed, not when
	it was prepared, and its period restarts from then
	"""

	# * Everything up to the delay is done over one connection, so it all
	# * happens within moments of the object being posted. Logging in again
	# * for each step would take a few seconds between the post and the prepare
	async with helpers.connect(nex.FRIEND_A) as client:
		post_info = await client.prepare_post_object(helpers.post_param(data_type=unique_data_type, size=helpers.POSTED_SIZE))
		data_id = post_info.data_id

		await client.complete_post_object(helpers.complete_post_param(data_id=data_id))
		update_info = await client.prepare_update_object(helpers.prepare_update_param(data_id=data_id, size=helpers.UPDATED_SIZE))
		before = await client.get_meta(helpers.get_meta_param(data_id=data_id))

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await complete(nex.FRIEND_A, data_id=data_id, version=update_info.version)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.period == before.period
	assert after.update_time.timestamp() - before.update_time.timestamp() >= helpers.TIMESTAMP_DELAY
	assert after.expire_time.timestamp() - before.expire_time.timestamp() >= helpers.TIMESTAMP_DELAY

async def test_a_persisted_object_stays_persisted(unique_data_type):
	"""
	Restarting the period does not apply to an object in a persistence
	slot, which never expires
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=0)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, persistence_init_param=persistence_init_param)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time.value() == constants.PERMANENT_DATE_TIME

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	There is no update to complete
	"""

	assert await complete_error(nex.FRIEND_A, data_id=data_id, version=2) == "DataStore::NotFound"

async def test_an_object_deleted_after_the_update_was_prepared(unique_data_type):
	"""
	The update goes with the object
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await complete_error(nex.FRIEND_A, data_id=data_id, version=version) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object which does not exist yet has had no update prepared
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)

	assert await complete_error(nex.FRIEND_A, data_id=data_id, version=2) == "DataStore::NotFound"

async def test_an_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	These objects can never have an update prepared, so there is none
	to complete
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.FRIEND_A, data_id=data_id, version=2) == "DataStore::NotFound"

# * version

@pytest.mark.parametrize("version", [
	pytest.param(0, id="zero"),
	pytest.param(1, id="the-current-version"),
	pytest.param(3, id="the-next-version")
])
async def test_a_version_which_was_never_handed_out(unique_data_type, version):
	"""
	Only a version handed out by PrepareUpdateObject can be completed,
	which here is only version 2
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	await prepare(nex.FRIEND_A, data_id)

	assert await complete_error(nex.FRIEND_A, data_id=data_id, version=version) == "DataStore::NotFound"

async def test_completing_two_versions_in_order(unique_data_type):
	"""
	Each completed update moves the object to its version
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	first = await prepare(nex.FRIEND_A, data_id, size=30)
	second = await prepare(nex.FRIEND_A, data_id, size=40)

	await complete(nex.FRIEND_A, data_id=data_id, version=first)
	await complete(nex.FRIEND_A, data_id=data_id, version=second)

	assert await helpers.version_of(data_id) == 3
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == 40

async def test_an_older_version_after_a_newer_one_changes_nothing(unique_data_type):
	"""
	An object never goes back to an older version. Completing one after
	a newer version is not an error, but the object is left alone
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	first = await prepare(nex.FRIEND_A, data_id, size=30)
	second = await prepare(nex.FRIEND_A, data_id, size=40)

	await complete(nex.FRIEND_A, data_id=data_id, version=second)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await complete(nex.FRIEND_A, data_id=data_id, version=first)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await helpers.version_of(data_id) == 3
	assert after.size == 40
	assert after.update_time.value() == before.update_time.value()

async def test_completing_the_same_version_twice(unique_data_type):
	"""
	Completing a version again is not an error, and counts as updating
	the object again, so the update time and expiry move again
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await helpers.version_of(data_id) == 2
	assert after.update_time.value() > before.update_time.value()
	assert after.expire_time.value() > before.expire_time.value()

# * success

async def test_a_failed_upload_changes_nothing(unique_data_type):
	"""
	Reporting a failed upload leaves the object as it was before the update
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)

	assert await helpers.version_of(data_id) == 1
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == helpers.POSTED_SIZE

async def test_a_failed_upload_can_still_be_completed(unique_data_type):
	"""
	The version stays handed out after a failure, so the upload can be
	retried and completed later
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)
	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert await helpers.version_of(data_id) == 2

async def test_reporting_a_failed_upload_twice(unique_data_type):
	"""
	Reporting a failed upload more than once is not an error, the
	object is simply left alone
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)
	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)

	assert await helpers.version_of(data_id) == 1

async def test_reporting_a_failure_after_completing(unique_data_type):
	"""
	Reporting a failed upload after the update was already completed is
	not an error, and does not undo the update
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)
	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)

	assert await helpers.version_of(data_id) == 2
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).size == helpers.UPDATED_SIZE

# * Who can complete the update
# *
# * An update belongs to whoever prepared it. Their update permission was
# * checked when it was prepared, and is not checked again

@pytest.mark.parametrize("preparer, completer", [
	pytest.param(nex.STRANGER, nex.FRIEND_A, id="the-owner-for-someone-else"),
	pytest.param(nex.FRIEND_A, nex.STRANGER, id="someone-with-update-permission-for-the-owner")
])
async def test_only_the_preparer_can_complete(unique_data_type, preparer, completer):
	"""
	Not even the owner, or someone else allowed to update the object,
	can complete an update somebody else prepared
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, delete_permission=update)
	version = await prepare(preparer, data_id)

	assert await complete_error(completer, data_id=data_id, version=version) == "DataStore::PermissionDenied"

async def test_losing_update_permission_after_preparing(unique_data_type):
	"""
	The preparer can still complete their update after the owner takes
	their update permission away
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, delete_permission=update)
	version = await prepare(nex.STRANGER, data_id)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATE_PERMISSION, delete_permission=helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE))
	await complete(nex.STRANGER, data_id=data_id, version=version)

	assert await helpers.version_of(data_id) == 2

async def test_an_update_prepared_with_the_update_password(unique_data_type):
	"""
	There is no password to give when completing, so someone who
	prepared the update with the update password can complete it without
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)
	version = await prepare(nex.STRANGER, data_id, update_password=password_info.update_password)

	await complete(nex.STRANGER, data_id=data_id, version=version)

	assert await helpers.version_of(data_id) == 2

# * The object status

async def test_an_object_under_review_stays_under_review(unique_data_type):
	"""
	Completing an update does not finish a review
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).status == constants.DataStatus.DATA_STATUS_PENDING

@pytest.mark.parametrize("status", [
	pytest.param(constants.DataStatus.DATA_STATUS_PENDING, id="put-under-review"),
	pytest.param(constants.DataStatus.DATA_STATUS_REJECTED, id="rejected")
])
async def test_a_status_change_after_preparing(unique_data_type, status):
	"""
	Someone other than the owner could not prepare an update now, but
	can still complete the one they prepared before, and the status is
	left as it is
	"""

	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, delete_permission=update)
	version = await prepare(nex.STRANGER, data_id)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=status)
	await complete(nex.STRANGER, data_id=data_id, version=version)

	assert await helpers.version_of(data_id) == 2
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).status == status

async def test_an_object_which_expired_after_preparing(unique_data_type):
	"""
	The update can still be completed, but restarting a period of 0
	leaves the object expired
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE)
	version = await prepare(nex.FRIEND_A, data_id)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)
	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * Notifications

@pytest.mark.parametrize("permission, recipients, notified", helpers.NOTIFIED_PERMISSIONS)
async def test_completing_notifies_the_recipients(unique_data_type, permission, recipients, notified):
	"""
	An object with DATA_FLAG_USE_NOTIFICATION_ON_UPDATE notifies its
	recipients once an update is completed, the same users a post
	notification goes to
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=access, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert await helpers.notified(data_id) == notified

async def test_a_failed_upload_does_not_notify(unique_data_type):
	"""
	The object was not updated, so its recipients are not notified
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version, success=False)

	assert await helpers.notified(data_id) == []

async def test_the_post_notification_flag_does_not_notify_on_update(unique_data_type):
	"""
	DATA_FLAG_USE_NOTIFICATION_ON_POST is only for posts, so the
	notification from the post is the only one there is
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST)
	before = await helpers.notification_ids(nex.STRANGER, data_id)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert len(before) == 1
	assert await helpers.notification_ids(nex.STRANGER, data_id) == before

async def test_the_update_notification_replaces_an_unread_post_notification(unique_data_type):
	"""
	A user only ever has one notification for an object, so the update
	notification takes the place of the post notification
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	flag = constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_POST | constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, flag=flag)
	before = await helpers.notification_ids(nex.STRANGER, data_id)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	after = await helpers.notification_ids(nex.STRANGER, data_id)

	assert len(before) == 1
	assert len(after) == 1
	assert after != before

@pytest.mark.parametrize("order", [
	pytest.param("same", id="the-same-version-again"),
	pytest.param("older", id="an-older-version-after-a-newer-one")
])
async def test_completing_again_notifies_again(unique_data_type, order):
	"""
	Every successful completion notifies the recipients again, even one
	which does not change the object
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE)
	first = await prepare(nex.FRIEND_A, data_id)
	second = await prepare(nex.FRIEND_A, data_id)

	if order == "same":
		await complete(nex.FRIEND_A, data_id=data_id, version=first)
		before = await helpers.notification_ids(nex.STRANGER, data_id)
		await complete(nex.FRIEND_A, data_id=data_id, version=first)
	else:
		await complete(nex.FRIEND_A, data_id=data_id, version=second)
		before = await helpers.notification_ids(nex.STRANGER, data_id)
		await complete(nex.FRIEND_A, data_id=data_id, version=first)

	after = await helpers.notification_ids(nex.STRANGER, data_id)

	assert len(before) == 1
	assert len(after) == 1
	assert after != before

async def test_the_preparer_is_notified_when_they_are_a_recipient(unique_data_type):
	"""
	Making the update does not leave the updater out of the notification
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	update = helpers.permission(permission=constants.Permission.PERMISSION_PUBLIC)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, delete_permission=update, flag=constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE)
	version = await prepare(nex.STRANGER, data_id)

	await complete(nex.STRANGER, data_id=data_id, version=version)

	assert await helpers.notified(data_id) == [nex.STRANGER]

async def test_an_object_under_review_notifies(unique_data_type):
	"""
	The recipients are notified even though the object is still pending
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	flag = constants.DataFlag.DATA_FLAG_USE_NOTIFICATION_ON_UPDATE | constants.DataFlag.DATA_FLAG_NEED_REVIEW
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, permission=permission, flag=flag)
	version = await prepare(nex.FRIEND_A, data_id)

	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	assert await helpers.notified(data_id) == [nex.STRANGER]

# * Completing an update does not count as referring to the object

async def test_completing_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move, even though the expiry does
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=helpers.POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	version = await prepare(nex.FRIEND_A, data_id)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await complete(nex.FRIEND_A, data_id=data_id, version=version)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
