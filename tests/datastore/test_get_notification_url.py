import pytest
from nintendo.nex import datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * What the notification file itself holds is not tested. At the time of
# * writing the file no longer exists on the file server, fetching it fails
# * with NoSuchKey, so there is nothing to check its contents against

async def get_notification_url(pid: int, previous_url: str = ""):
	"""
	Gets where the users notification file is, as the given user
	"""

	param = datastore.DataStoreGetNotificationUrlParam()
	param.previous_url = previous_url

	async with helpers.connect(pid) as client:
		return await client.get_notification_url(param)

async def unread(pid: int) -> list[tuple[int, int]]:
	"""
	The notification ID and data ID of every unread notification the user has
	"""

	async with helpers.connect(pid) as client:
		response = await client.get_new_arrived_notifications(helpers.new_arrived_notifications_param())

	return [(notification.notification_id, notification.data_id) for notification in response.result]

# * What comes back

async def test_returns_where_the_notification_file_is():
	"""
	Basic functionality test
	"""

	info = await get_notification_url(nex.STRANGER)

	assert info.url != ""
	assert info.key != ""
	assert info.query != ""

async def test_the_key_is_the_same_on_every_call():
	"""
	Each user has one notification file, so the key always points at it
	"""

	first = await get_notification_url(nex.STRANGER)
	second = await get_notification_url(nex.STRANGER)

	assert second.url == first.url
	assert second.key == first.key

async def test_each_user_has_their_own_file():
	"""
	Every user has a notification file of their own
	"""

	assert (await get_notification_url(nex.STRANGER)).key != (await get_notification_url(nex.FRIEND_A)).key

# * previous_url

@pytest.mark.parametrize("previous_url", [
	pytest.param(None, id="the-previous-url"),
	pytest.param("not a url", id="not-a-url")
])
async def test_previous_url_is_ignored(previous_url):
	"""
	Whatever the client last had, the same file is pointed at, and nothing
	about the previous URL is checked
	"""

	# TODO - It's not clear if this is intentional or just part of Bank/the Nintendo Network shutdown

	first = await get_notification_url(nex.STRANGER)

	if previous_url is None:
		previous_url = first.url + first.key + first.query

	again = await get_notification_url(nex.STRANGER, previous_url)

	assert again.url == first.url
	assert again.key == first.key

# * The notification it creates

async def test_creates_a_notification_with_no_object():
	"""
	Each call gives the caller a notification whose data ID is
	INVALID_DATAID
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await get_notification_url(nex.STRANGER)

	assert [data_id for _, data_id in await unread(nex.STRANGER)] == [constants.INVALID_DATAID]

async def test_calling_again_replaces_that_notification():
	"""
	As with every other notification, the user only ever has one for the
	same data ID, so a new one takes the place of the old
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await get_notification_url(nex.STRANGER)

	[(first, _)] = await unread(nex.STRANGER)

	await get_notification_url(nex.STRANGER)

	[(second, data_id)] = await unread(nex.STRANGER)

	assert data_id == constants.INVALID_DATAID
	assert second > first

async def test_only_the_caller_gets_the_notification():
	"""
	Nobody else is notified
	"""

	await helpers.read_every_notification(nex.FRIEND_B)
	await get_notification_url(nex.STRANGER)

	assert await unread(nex.FRIEND_B) == []
