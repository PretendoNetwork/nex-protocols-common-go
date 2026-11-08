import pytest

import nex
from datastore import helpers

pytestmark = pytest.mark.anyio

async def get_new_arrived_notifications(pid: int, **fields):
	"""
	Gets the users unread notifications. Each field is a field of the param
	"""

	async with helpers.connect(pid) as client:
		return await client.get_new_arrived_notifications(helpers.new_arrived_notifications_param(**fields))

def entries(response) -> list[tuple[int, int]]:
	"""
	The notification ID and data ID of every notification which came back
	"""

	return [(notification.notification_id, notification.data_id) for notification in response.result]

# * What comes back
# *
# * Every test reads the strangers notifications first, so it starts with
# * none

async def test_a_notification_points_at_its_object(unique_data_type):
	"""
	Each notification has its own ID, and the data ID of the object it
	is for
	"""

	await helpers.read_every_notification(nex.STRANGER)
	[data_id] = await helpers.notify(nex.STRANGER, 1, unique_data_type)

	response = await get_new_arrived_notifications(nex.STRANGER)

	assert [data_id for _, data_id in entries(response)] == [data_id]
	assert response.result[0].notification_id != 0
	assert not response.has_next

async def test_nothing_has_arrived():
	"""
	A user with no notifications gets an empty list, which is not an error
	"""

	await helpers.read_every_notification(nex.STRANGER)
	response = await get_new_arrived_notifications(nex.STRANGER)

	assert response.result == []
	assert not response.has_next

async def test_notifications_come_oldest_first(unique_data_type):
	"""
	Notifications are in the order they arrived, which is also the order
	of their IDs
	"""

	await helpers.read_every_notification(nex.STRANGER)
	data_ids = await helpers.notify(nex.STRANGER, 3, unique_data_type)
	notification_ids = [notification_id for notification_id, _ in entries(await get_new_arrived_notifications(nex.STRANGER))]

	assert [data_id for _, data_id in entries(await get_new_arrived_notifications(nex.STRANGER))] == data_ids
	assert notification_ids == sorted(notification_ids)

async def test_each_user_only_gets_their_own(unique_data_type):
	"""
	A notification only arrives for the user it was sent to
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.read_every_notification(nex.FRIEND_B)
	await helpers.notify(nex.STRANGER, 1, unique_data_type)

	assert (await get_new_arrived_notifications(nex.FRIEND_B)).result == []

async def test_getting_notifications_does_not_remove_them(unique_data_type):
	"""
	Only last_notification_id removes notifications, so getting them from
	0 again gives the same ones back
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, 2, unique_data_type)

	first = entries(await get_new_arrived_notifications(nex.STRANGER))

	assert entries(await get_new_arrived_notifications(nex.STRANGER)) == first

# * last_notification_id
# *
# * Passing a notification ID marks every notification up to and including
# * it as read, which removes them from future results

async def test_reading_up_to_a_notification(unique_data_type):
	"""
	The notification and every one before it are removed, and only the
	ones after it come back, now and later
	"""

	await helpers.read_every_notification(nex.STRANGER)
	data_ids = await helpers.notify(nex.STRANGER, 3, unique_data_type)
	notification_ids = [notification_id for notification_id, _ in entries(await get_new_arrived_notifications(nex.STRANGER))]

	response = await get_new_arrived_notifications(nex.STRANGER, last_notification_id=notification_ids[1])

	assert entries(response) == [(notification_ids[2], data_ids[2])]
	assert entries(await get_new_arrived_notifications(nex.STRANGER)) == [(notification_ids[2], data_ids[2])]

async def test_a_last_notification_id_before_every_notification(unique_data_type):
	"""
	Nothing comes before it, so nothing is removed
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, 3, unique_data_type)
	before = entries(await get_new_arrived_notifications(nex.STRANGER))

	assert entries(await get_new_arrived_notifications(nex.STRANGER, last_notification_id=before[0][0] - 1)) == before

async def test_a_last_notification_id_past_every_notification(unique_data_type):
	"""
	Every notification is removed, and nothing comes back
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, 3, unique_data_type)
	newest = entries(await get_new_arrived_notifications(nex.STRANGER))[-1][0]

	assert (await get_new_arrived_notifications(nex.STRANGER, last_notification_id=newest + 1000)).result == []
	assert (await get_new_arrived_notifications(nex.STRANGER)).result == []

async def test_reading_ahead_does_not_hide_later_notifications(unique_data_type):
	"""
	Only notifications which already exist are removed. One which arrives
	afterwards comes back even though its ID is below the one read up to
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, 1, unique_data_type)
	newest = entries(await get_new_arrived_notifications(nex.STRANGER))[-1][0]

	await get_new_arrived_notifications(nex.STRANGER, last_notification_id=newest + 1000)
	[data_id] = await helpers.notify(nex.STRANGER, 1, unique_data_type)
	[(notification_id, arrived)] = entries(await get_new_arrived_notifications(nex.STRANGER))

	assert arrived == data_id
	assert notification_id < newest + 1000

# * limit

@pytest.mark.parametrize("limit, count, returned", [
	pytest.param(0, 21, 20, id="zero"),
	pytest.param(99, 100, 99, id="the-largest-limit"),
	pytest.param(100, 21, 20, id="past-the-largest-limit")
])
async def test_the_limit(unique_data_type, limit, count, returned):
	"""
	Up to 99 notifications can be asked for. A limit of 0, or of more than
	99, gives 20
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, count, unique_data_type)

	response = await get_new_arrived_notifications(nex.STRANGER, limit=limit)

	assert len(response.result) == returned
	assert response.has_next

@pytest.mark.parametrize("limit, has_next", [
	pytest.param(0, False, id="zero"),
	pytest.param(20, False, id="exactly-enough"),
	pytest.param(19, True, id="one-short")
])
async def test_has_next(unique_data_type, limit, has_next):
	"""
	has_next says whether there are more notifications than came back
	"""

	await helpers.read_every_notification(nex.STRANGER)
	await helpers.notify(nex.STRANGER, 20, unique_data_type)

	assert (await get_new_arrived_notifications(nex.STRANGER, limit=limit)).has_next == has_next

async def test_paging_through_every_notification(unique_data_type):
	"""
	Passing the last notification of one page as last_notification_id
	gets the next page, until every notification has come back once
	"""

	await helpers.read_every_notification(nex.STRANGER)
	data_ids = await helpers.notify(nex.STRANGER, 25, unique_data_type)

	arrived = []
	last_notification_id = 0

	while True:
		response = await get_new_arrived_notifications(nex.STRANGER, last_notification_id=last_notification_id, limit=10)
		arrived += [data_id for _, data_id in entries(response)]

		if not response.has_next:
			break

		last_notification_id = response.result[-1].notification_id

	assert arrived == data_ids
