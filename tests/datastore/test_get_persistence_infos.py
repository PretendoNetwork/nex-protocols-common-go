import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT_0 = 0
SLOT_1 = 1
EMPTY_SLOT = 15

# * Reading a slot is gated by the access permission of the object which
# * is in it, so each permission is tried against somebody it lets in and
# * somebody it keeps out. A public object keeps nobody out, so it has no
# * denied user
ACCESS_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, None, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, nex.STRANGER, id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, nex.FRIEND_B, id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, nex.STRANGER, id="specified-friend")
]

async def get_persistence_infos(pid: int, owner_id: int, slots: list[int]):
	"""
	Reads several of a users persistence slots
	"""

	async with helpers.connect(pid) as client:
		return await client.get_persistence_infos(owner_id, slots)

async def get_persistence_infos_error(pid: int, owner_id: int, slots: list[int]) -> str:
	"""
	Reads several slots and expects an error to be thrown, returning
	the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_persistence_infos(owner_id, slots)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetPersistenceInfos did not throw an error")

def slots_of(response) -> list[int]:
	"""
	The slot of every info which came back
	"""

	return [info.slot_id for info in response.infos]

# * What the slots hold

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_returns_an_info_for_every_slot(unique_data_type, post):
	"""
	Every slot in the list gets its own entry back
	"""

	first = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	second = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_1, data_type=unique_data_type)

	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0, SLOT_1])

	assert helpers.result_names(response.results) == [None, None]
	assert helpers.data_ids_of(response) == [first, second]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_the_slots_come_back_in_the_order_they_were_asked_for(unique_data_type, post):
	"""
	The caller matches the results by position
	"""

	first = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	second = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_1, data_type=unique_data_type)

	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_1, SLOT_0])

	assert slots_of(response) == [SLOT_1, SLOT_0]
	assert helpers.data_ids_of(response) == [second, first]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_empty_slot_only_fails_its_own_entry(unique_data_type, post):
	"""
	Empty slots fail on their own, rather than failing the whole request
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0, EMPTY_SLOT])

	assert helpers.result_names(response.results) == [None, "DataStore::NotFound"]
	assert response.infos[SLOT_0].data_id == data_id
	assert response.infos[SLOT_1].data_id == 0

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_the_newest_object_wins(unique_data_type, post):
	"""
	Putting another object in a slot replaces the one which was there
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	second = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)

	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0])

	assert helpers.result_names(response.results) == [None]
	assert helpers.data_ids_of(response) == [second]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_deleting_the_object_empties_the_slot(unique_data_type, post):
	"""
	Deleting a persisted object leaves the slot with nothing in it
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0])

	assert helpers.result_names(response.results) == ["DataStore::NotFound"]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_the_same_slot_twice(unique_data_type, post):
	"""
	Reading the same slot multiple times is supported
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0, SLOT_0])

	assert helpers.result_names(response.results) == [None, None]
	assert helpers.data_ids_of(response) == [data_id, data_id]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_no_slots_returns_every_slot(unique_data_type, post):
	"""
	An empty list of slots returns all of the users slots, including
	ones which are not filled
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [])

	assert slots_of(response) == list(range(constants.NUM_PERSISTENCE_SLOT))
	assert response.infos[SLOT_0].data_id == data_id
	assert response.results[SLOT_0].is_success()
	assert response.results[EMPTY_SLOT].is_error()

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_more_entries_than_there_are_slots(unique_data_type, post):
	"""
	Nothing stops the same slot being listed over and over, so a list
	longer than the number of slots is still fine
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	slots = [SLOT_0] * (constants.NUM_PERSISTENCE_SLOT + 1)

	response = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, slots)

	assert helpers.result_names(response.results) == [None] * len(slots)

# * owner_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_owner_of_zero_means_the_caller(unique_data_type, post):
	"""
	Asking about nobody in particular reads the callers own slots
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	response = await get_persistence_infos(nex.FRIEND_A, 0, [SLOT_0])

	assert helpers.result_names(response.results) == [None]
	assert response.infos[SLOT_0].data_id == data_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_owner_of_zero_with_no_slots(unique_data_type, post):
	"""
	Using no PID and an empty list returns all of the callers slots
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	response = await get_persistence_infos(nex.FRIEND_A, 0, [])

	assert slots_of(response) == list(range(constants.NUM_PERSISTENCE_SLOT))
	assert [info.owner_id for info in response.infos] == [nex.FRIEND_A] * constants.NUM_PERSISTENCE_SLOT
	assert response.infos[SLOT_0].data_id == data_id
	assert response.results[SLOT_0].is_success()
	assert response.results[EMPTY_SLOT].is_error()

async def test_unknown_owner():
	"""
	A user who has never persisted anything has no slots to read, and
	that fails per slot rather than failing the request
	"""

	response = await get_persistence_infos(nex.FRIEND_A, 1234, [SLOT_0])

	assert helpers.result_names(response.results) == ["DataStore::NotFound"]

# * slot_ids
# *
# * Slots which are not slots at all

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_PERSISTENCE_SLOT, id="one-past-the-last-slot"),
	pytest.param(constants.INVALID_PERSISTENCE_SLOT_ID, id="invalid-slot")
])
async def test_an_invalid_slot_fails_the_whole_request(slot):
	"""
	A slot which cannot exist is a bad request, which throws instead of failing only its own entry
	"""

	assert await get_persistence_infos_error(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0, slot]) == "DataStore::InvalidArgument"

# * Who may read somebody elses slots

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
@pytest.mark.parametrize("permission, recipients, allowed, denied", ACCESS_PERMISSIONS)
async def test_the_access_permission_of_the_object(unique_data_type, post, permission, recipients, allowed, denied):
	"""
	Reading a slot is gated by the access permission of the object in it,
	and a slot which is closed fails on its own
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type, permission=access)

	opened = await get_persistence_infos(allowed, nex.FRIEND_A, [SLOT_0])

	assert helpers.result_names(opened.results) == [None]
	assert opened.infos[SLOT_0].data_id == data_id

	if denied is not None:
		closed = await get_persistence_infos(denied, nex.FRIEND_A, [SLOT_0])

		assert helpers.result_names(closed.results) == ["DataStore::PermissionDenied"]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
@pytest.mark.parametrize("caller", [
	pytest.param(nex.FRIEND_B, id="a-friend"),
	pytest.param(nex.STRANGER, id="a-stranger")
])
async def test_a_private_object_is_closed_to_everybody(unique_data_type, post, caller):
	"""
	A private object lets nobody in at all
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type, permission=permission)

	response = await get_persistence_infos(caller, nex.FRIEND_A, [SLOT_0])

	assert helpers.result_names(response.results) == ["DataStore::PermissionDenied"]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_object_under_review(unique_data_type, post):
	"""
	An object waiting to be reviewed reports as such, but is viewable to
	its owner like normal
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	mine = await get_persistence_infos(nex.FRIEND_A, nex.FRIEND_A, [SLOT_0])
	theirs = await get_persistence_infos(nex.STRANGER, nex.FRIEND_A, [SLOT_0])

	assert mine.infos[SLOT_0].data_id == data_id
	assert helpers.result_names(theirs.results) == ["DataStore::UnderReviewing"]
