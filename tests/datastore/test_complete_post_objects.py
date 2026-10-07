import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def complete(pid: int, data_ids: list[int]) -> None:
	"""
	Completes the uploads of several objects as the given user
	"""

	async with helpers.connect(pid) as client:
		await client.complete_post_objects(data_ids)

async def complete_error(pid: int, data_ids: list[int]) -> str:
	"""
	Completes the uploads of several objects and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.complete_post_objects(data_ids)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::CompletePostObjects did not throw an error")

async def test_completes_several_uploads(unique_data_type):
	"""
	Every object in the list is completed
	"""

	first = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)
	second = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	await complete(nex.FRIEND_A, [first, second])

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=first)).data_id == first
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=second)).data_id == second

async def test_no_data_ids():
	"""
	Completing nothing at all is not an error
	"""

	await complete(nex.FRIEND_A, [])

async def test_completing_twice(unique_data_type):
	"""
	Objects which were already completed cannot be completed again.
	Unlike CompletePostObject, this method uses OperationNotAllowed
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)
	await complete(nex.FRIEND_A, [data_id])

	assert await complete_error(nex.FRIEND_A, [data_id]) == "DataStore::OperationNotAllowed"

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UINT64_MAX, id="uint64-max")
])
async def test_unknown_data_id(data_id):
	"""
	Completing an object which does not exist fails. Unlike
	CompletePostObject, this method uses NotFound
	"""

	assert await complete_error(nex.FRIEND_A, [data_id]) == "DataStore::NotFound"

async def test_only_the_owner_can_complete(unique_data_type):
	"""
	Only the object owner can complete its upload
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.STRANGER, [data_id]) == "DataStore::PermissionDenied"

async def test_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	Objects which do not use the file server are already available,
	so they cannot be completed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.FRIEND_A, [data_id]) == "DataStore::OperationNotAllowed"

async def test_too_many_data_ids():
	"""
	No more than BATCH_PROCESSING_CAPACITY objects can be completed at once
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * (constants.BATCH_PROCESSING_CAPACITY + 1)

	assert await complete_error(nex.FRIEND_A, data_ids) == "DataStore::OverCapacity"

async def test_the_most_data_ids_allowed():
	"""
	Exactly BATCH_PROCESSING_CAPACITY objects is allowed
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * constants.BATCH_PROCESSING_CAPACITY

	assert await complete_error(nex.FRIEND_A, data_ids) == "DataStore::NotFound"

# * Rollback tests
# *
# * When any object in the list cannot be completed, none of the
# * objects in that list are completed

async def test_nothing_is_completed_when_an_object_does_not_exist(unique_data_type):
	"""
	An unknown object in the list leaves the other objects alone
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.FRIEND_A, [data_id, helpers.UNKNOWN_DATA_ID]) == "DataStore::NotFound"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_nothing_is_completed_when_an_object_is_already_completed(unique_data_type):
	"""
	An object which does not use the file server is already completed,
	so it leaves the other objects in the list alone
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)
	meta_binary_data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.FRIEND_A, [data_id, meta_binary_data_id]) == "DataStore::OperationNotAllowed"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_nothing_is_completed_when_an_object_belongs_to_someone_else(unique_data_type):
	"""
	Someone elses object in the list leaves the callers own objects alone
	"""

	mine = await helpers.prepare_post_object(nex.STRANGER, data_type=unique_data_type)
	theirs = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.STRANGER, [mine, theirs]) == "DataStore::PermissionDenied"
	assert await helpers.get_meta_error(nex.STRANGER, data_id=mine) == "DataStore::NotFound"

async def test_the_same_object_twice_in_one_call(unique_data_type):
	"""
	The second time an object appears in the list it has already been
	completed by the first, so the whole call fails and the object is
	left alone. It can still be completed afterwards
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.FRIEND_A, [data_id, data_id]) == "DataStore::OperationNotAllowed"
	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

	await complete(nex.FRIEND_A, [data_id])

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).data_id == data_id
