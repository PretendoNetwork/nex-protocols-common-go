import anyio
import pytest
from nintendo.nex import datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

POSTED_SIZE = 10

async def get_specific_meta(pid: int, data_ids: list[int]) -> list[datastore.DataStoreSpecificMetaInfo]:
	"""
	Gets the minimal metadata of several objects as the given user
	"""

	async with helpers.connect(pid) as client:
		param = datastore.DataStoreGetSpecificMetaParam()
		param.data_ids = data_ids

		return await client.get_specific_meta(param)

async def returned_data_ids(pid: int, data_ids: list[int]) -> list[int]:
	"""
	The data ID of every entry which came back
	"""

	return [info.data_id for info in await get_specific_meta(pid, data_ids)]

# * What an entry holds

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_an_entry_describes_the_object(unique_data_type, post):
	"""
	Basic functionality test
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	[info] = await get_specific_meta(nex.FRIEND_A, [data_id])

	assert info.data_id == data_id
	assert info.owner_id == nex.FRIEND_A
	assert info.size == POSTED_SIZE
	assert info.data_type == unique_data_type
	assert info.version == 1

async def test_the_size_of_an_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	These objects have no file, and GetMeta gives their size as 0, but
	here the size is the length of their meta binary
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"0123456789")
	[info] = await get_specific_meta(nex.FRIEND_A, [data_id])

	assert info.size == 10

async def test_an_object_posted_with_an_empty_meta_binary(unique_data_type):
	"""
	The length of a meta binary which was empty from the start comes out
	as UINT32_MAX, rather than as 0
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"")
	[info] = await get_specific_meta(nex.FRIEND_A, [data_id])

	assert info.size == helpers.UINT32_MAX

# * data_ids

async def test_no_data_ids():
	"""
	Asking for nothing is not an error
	"""

	assert await get_specific_meta(nex.FRIEND_A, []) == []

async def test_the_entries_are_sorted_by_data_id(unique_data_type):
	"""
	The entries do not come back in the order they were asked for
	"""

	data_ids = [await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE) for _ in range(3)]

	assert await returned_data_ids(nex.FRIEND_A, list(reversed(data_ids))) == sorted(data_ids)

async def test_a_data_id_asked_for_twice_comes_back_once(unique_data_type):
	"""
	Each object has one entry, no matter however many times it is asked for
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)

	assert await returned_data_ids(nex.FRIEND_A, [data_id, data_id]) == [data_id]

async def test_there_is_no_capacity(unique_data_type):
	"""
	Unlike the other batch methods, more than BATCH_PROCESSING_CAPACITY
	objects can be asked for at once
	"""

	async with helpers.connect(nex.FRIEND_A) as client:
		data_ids = [await client.post_meta_binary(helpers.post_param(data_type=unique_data_type)) for _ in range(constants.BATCH_PROCESSING_CAPACITY + 1)]

	assert await returned_data_ids(nex.FRIEND_A, data_ids) == data_ids

@pytest.mark.parametrize("missing", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing_is_left_out(unique_data_type, missing):
	"""
	There are no per entry results, so an object which cannot be found
	simply has no entry, and the rest still come back
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)

	assert await returned_data_ids(nex.FRIEND_A, [data_id, missing]) == [data_id]

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This is only left out because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await get_specific_meta(nex.FRIEND_A, [999999]) == []

async def test_an_object_with_a_size_of_0_is_left_out(unique_data_type):
	"""
	An object whose size is 0 has no entry, even though it exists
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=0)

	assert await get_specific_meta(nex.FRIEND_A, [data_id]) == []

async def test_a_meta_binary_changed_to_empty_is_left_out(unique_data_type):
	"""
	Unlike one which was empty from the start, a meta binary which was
	emptied counts as a size of 0
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"meta")
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_METABINARY, meta_binary=b"")

	assert await get_specific_meta(nex.FRIEND_A, [data_id]) == []

async def test_a_deleted_object_still_comes_back(unique_data_type):
	"""
	Deleting an object does not stop its entry from being returned
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await returned_data_ids(nex.FRIEND_A, [data_id]) == [data_id]

async def test_an_object_which_was_never_completed_still_comes_back(unique_data_type):
	"""
	The object does not exist yet anywhere else, but it has an entry here
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)

	assert await returned_data_ids(nex.FRIEND_A, [data_id]) == [data_id]

async def test_an_expired_object_still_comes_back(unique_data_type):
	"""
	An object with a period of 0 has expired, but it has an entry here
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	assert await returned_data_ids(nex.FRIEND_A, [data_id]) == [data_id]

# * The objects access permission and status
# *
# * Neither is checked, anyone can get the entry of any object

@pytest.mark.parametrize("permission, recipients", helpers.PERMISSIONS)
async def test_the_access_permission_is_not_checked(unique_data_type, permission, recipients):
	"""
	Even an object nobody else may read has an entry for everyone
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, permission=access)

	assert await returned_data_ids(nex.STRANGER, [data_id]) == [data_id]

@pytest.mark.parametrize("status", [
	pytest.param(constants.DataStatus.DATA_STATUS_PENDING, id="pending"),
	pytest.param(constants.DataStatus.DATA_STATUS_REJECTED, id="rejected")
])
async def test_the_status_is_not_checked(unique_data_type, status):
	"""
	An object which is under review or rejected has an entry for everyone
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=status)

	assert await returned_data_ids(nex.STRANGER, [data_id]) == [data_id]

# * Reading the metadata does not count as referring to the object

async def test_reading_the_metadata_does_not_refer_to_the_object(unique_data_type):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=POSTED_SIZE, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await get_specific_meta(nex.STRANGER, [data_id])

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
