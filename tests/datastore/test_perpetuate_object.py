import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT_0 = 0
SLOT_1 = 1
EMPTY_SLOT = 15

async def perpetuate(pid: int, slot: int, data_id: int, delete_last_object: bool = False) -> None:
	"""
	Puts an object into one of the users persistence slots
	"""

	async with helpers.connect(pid) as client:
		await client.perpetuate_object(slot, data_id, delete_last_object)

async def perpetuate_error(pid: int, slot: int, data_id: int, delete_last_object: bool = False) -> str:
	"""
	Puts an object into a persistence slot and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.perpetuate_object(slot, data_id, delete_last_object)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PerpetuateObject did not throw an error")

async def empty_slot(pid: int, slot: int) -> None:
	"""
	Makes sure a slot has nothing in it
	"""

	async with helpers.connect(pid) as client:
		try:
			await client.unperpetuate_object(slot, False)
		except common.RMCError:
			pass

# * Putting an object into a slot

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_perpetuates_an_object(unique_data_type, post):
	"""
	An object which was posted without a persistence slot can be put
	into one afterwards
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_perpetuating_the_same_object_twice(unique_data_type, post):
	"""
	Putting an object into the slot it is already in is allowed, and
	does not delete the object even though it is its own last object
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)
	await perpetuate(nex.FRIEND_A, SLOT_0, data_id, delete_last_object=True)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == data_id
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_object_can_only_be_in_one_slot(unique_data_type, post):
	"""
	An object which is already in a slot cannot be put into another
	one. The call doesn't fail but leaves both slots as they were
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await empty_slot(nex.FRIEND_A, SLOT_1) # * Make sure the target slot is empty
	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)
	await perpetuate(nex.FRIEND_A, SLOT_1, data_id)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == data_id
	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT_1) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("delete_last_object", [
	pytest.param(False, id="keeping-the-old-object"),
	pytest.param(True, id="deleting-the-old-object"),
])
async def test_an_object_in_a_slot_cannot_take_over_another_one(unique_data_type, post, delete_last_object):
	"""
	Moving into an empty slot quietly does nothing, but moving into a
	slot which is taken is a failure
	"""

	mover = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)
	sitter = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_1, data_type=unique_data_type)

	assert await perpetuate_error(nex.FRIEND_A, SLOT_1, mover, delete_last_object) == "DataStore::OperationNotAllowed"

	# * Neither slot moved, and nothing was deleted on the way
	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == mover
	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_1)).data_id == sitter
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=mover)).data_id == mover
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=sitter)).data_id == sitter

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_object_with_no_slot_can_take_over_one(unique_data_type, post):
	"""
	An object not in a slot can take over an occupied slot. The rejection in
	test_an_object_in_a_slot_cannot_take_over_another_one is about moving an
	object from one slot to another, not about overtaking an occupied slot
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, SLOT_1, data_type=unique_data_type)

	loose = await post(nex.FRIEND_A, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_1, loose)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_1)).data_id == loose

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_object_under_review(unique_data_type, post):
	"""
	An object waiting to be reviewed can still be put into a slot
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == data_id

# * An object in a slot is exempt from expiry

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_perpetuating_stops_the_object_expiring(unique_data_type, post):
	"""
	An object in a slot is given the permanent date, so it is kept
	until something takes it out of the slot
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time.value() != constants.PERMANENT_DATE_TIME

	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).expire_time.value() == constants.PERMANENT_DATE_TIME

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_perpetuating_leaves_the_reference_count_alone(unique_data_type, post):
	"""
	Putting an object into a slot is not the same as referring to it,
	so the reference count and referred time are untouched
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await perpetuate(nex.FRIEND_A, SLOT_0, data_id)
	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_object_let_out_of_a_slot_expires_again(unique_data_type, post):
	"""
	An object which is replaced but kept goes back to expiring
	normally, rather than keeping the permanent date
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_0, first)
	await perpetuate(nex.FRIEND_A, SLOT_0, second, delete_last_object=False)

	expire_time = (await helpers.get_meta(nex.FRIEND_A, data_id=first)).expire_time

	assert expire_time.value() != constants.PERMANENT_DATE_TIME
	assert expire_time.value() > (await helpers.get_meta(nex.FRIEND_A, data_id=second)).create_time.value()

# * Replacing whatever was in the slot already

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_replacing_an_object_deletes_the_old_one(unique_data_type, post):
	"""
	The object which was in the slot is deleted when delete_last_object is set
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_0, first)
	await perpetuate(nex.FRIEND_A, SLOT_0, second, delete_last_object=True)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == second
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=first) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_replacing_an_object_can_keep_the_old_one(unique_data_type, post):
	"""
	The object which was in the slot is kept when delete_last_object is not set
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_0, first)
	await perpetuate(nex.FRIEND_A, SLOT_0, second, delete_last_object=False)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == second
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).data_id == first

# * Objects which cannot be put into a slot

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(999999, id="the-system-range"),
	pytest.param(helpers.UINT64_MAX, id="uint64-max"),
])
async def test_unknown_data_id(data_id):
	"""
	Objects which do not exist cannot be persisted
	"""

	assert await perpetuate_error(nex.FRIEND_A, SLOT_0, data_id) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet, so
	it cannot be persisted
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await perpetuate_error(nex.FRIEND_A, SLOT_0, data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects cannot be persisted
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await perpetuate_error(nex.FRIEND_A, SLOT_0, data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_only_the_owner_can_perpetuate(unique_data_type, post):
	"""
	Only the object owner can put it into a persistence slot
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await perpetuate_error(nex.STRANGER, SLOT_0, data_id) == "DataStore::PermissionDenied"
	assert await perpetuate_error(nex.FRIEND_B, SLOT_0, data_id) == "DataStore::PermissionDenied"

# * A data ID of 0 names no object

async def test_a_data_id_of_zero_into_an_empty_slot():
	"""
	The slot is left empty rather than the request being refused
	"""

	await perpetuate(nex.FRIEND_A, EMPTY_SLOT, constants.INVALID_DATAID)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, EMPTY_SLOT) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_data_id_of_zero_into_a_filled_slot(unique_data_type, post):
	"""
	The slot keeps what it had
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT_0, data_type=unique_data_type)

	assert await perpetuate_error(nex.FRIEND_A, SLOT_0, constants.INVALID_DATAID) == "DataStore::NotFound"
	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == data_id

# * Slots which are not slots at all

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_PERSISTENCE_SLOT, id="one-past-the-last-slot"),
	pytest.param(constants.INVALID_PERSISTENCE_SLOT_ID, id="invalid-slot"),
])
async def test_invalid_slot(unique_data_type, slot):
	"""
	Only slots 0 to NUM_PERSISTENCE_SLOT - 1 can be used
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await perpetuate_error(nex.FRIEND_A, slot, data_id) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_slots_are_per_user(unique_data_type, post):
	"""
	Filling a slot only affects the callers own slots
	"""

	mine = await post(nex.FRIEND_A, data_type=unique_data_type)
	theirs = await post(nex.STRANGER, data_type=unique_data_type)

	await perpetuate(nex.FRIEND_A, SLOT_0, mine)
	await perpetuate(nex.STRANGER, SLOT_0, theirs)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT_0)).data_id == mine
	assert (await helpers.get_persistence_info(nex.STRANGER, nex.STRANGER, SLOT_0)).data_id == theirs
