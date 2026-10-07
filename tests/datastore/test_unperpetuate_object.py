import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT_0 = 0
EMPTY_SLOT = 15

async def unperpetuate(pid: int, slot: int, delete_last_object: bool = False) -> None:
	"""
	Empties one of the users persistence slots
	"""

	async with helpers.connect(pid) as client:
		await client.unperpetuate_object(slot, delete_last_object)

async def unperpetuate_error(pid: int, slot: int, delete_last_object: bool = False) -> str:
	"""
	Empties a persistence slot and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.unperpetuate_object(slot, delete_last_object)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::UnperpetuateObject did not throw an error")

# * Emptying a slot, and what happens to the object which was in it

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_empties_the_slot(unique_data_type, post):
	"""
	The slot is simply emptied when delete_last_object=False
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await unperpetuate(nex.FRIEND_A, SLOT_0, delete_last_object=False)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT_0) == "DataStore::NotFound"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id

# * persistence_slot_id
# *
# * Slots with nothing in them

@pytest.mark.parametrize("delete_last_object", [
	pytest.param(False, id="keeping-the-object"),
	pytest.param(True, id="deleting-the-object")
])
async def test_empty_slot(delete_last_object):
	"""
	A slot which has no object in it cannot be emptied
	"""

	assert await unperpetuate_error(nex.FRIEND_A, EMPTY_SLOT, delete_last_object) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_emptying_a_slot_twice(unique_data_type, post):
	"""
	The slot is already empty the second time
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await unperpetuate(nex.FRIEND_A, SLOT_0)

	assert await unperpetuate_error(nex.FRIEND_A, SLOT_0) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_a_slot_whose_object_was_deleted(unique_data_type, post):
	"""
	Deleting the object empties the slot on its own
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await unperpetuate_error(nex.FRIEND_A, SLOT_0) == "DataStore::NotFound"

# * persistence_slot_id, which is not a slot at all

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_PERSISTENCE_SLOT, id="one-past-the-last-slot"),
	pytest.param(constants.INVALID_PERSISTENCE_SLOT_ID, id="invalid-slot")
])
async def test_invalid_slot(slot):
	"""
	Only slots 0 to NUM_PERSISTENCE_SLOT - 1 can be used
	"""

	assert await unperpetuate_error(nex.FRIEND_A, slot) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_slots_are_per_user(unique_data_type, post):
	"""
	Emptying a slot only affects the callers own slots
	"""

	mine = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await helpers.post_persisted(post, nex.STRANGER, SLOT_0, data_type=unique_data_type)

	await unperpetuate(nex.STRANGER, SLOT_0)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == mine

# * delete_last_object

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_can_delete_the_object_too(unique_data_type, post):
	"""
	The object which was in the slot is deleted when delete_last_object=True
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	await unperpetuate(nex.FRIEND_A, SLOT_0, delete_last_object=True)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT_0) == "DataStore::NotFound"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_the_object_expires_again(unique_data_type, post):
	"""
	An object in a slot is kept forever, so letting it out has to put
	it back on a normal expiry
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time.value() == constants.PERMANENT_DATE_TIME

	await unperpetuate(nex.FRIEND_A, SLOT_0, delete_last_object=False)
	expire_time = (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time

	assert expire_time.value() != constants.PERMANENT_DATE_TIME
	assert expire_time.value() > (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).create_time.value()

# * The object status

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_object_under_review(unique_data_type, post):
	"""
	An object waiting to be reviewed can be let out of its slot like any other
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	await unperpetuate(nex.FRIEND_A, SLOT_0, delete_last_object=False)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT_0) == "DataStore::NotFound"
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id
