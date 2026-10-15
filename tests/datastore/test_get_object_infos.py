import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * DISCLAIMER, THESE TESTS ASSUME THE WIKI IS CORRECT AND HAVE NOT BEEN
# * CHECKED AGAINST A REAL SERVER. THE ONLY LIVE NEX SERVER ON NINTENDO
# * NETWORK AT THE TIME OF WRITING IS POKEMON BANK, WHICH HAS CUSTOM RMCs
# * FROM 40+, SO THIS COULD NOT BE TESTED. ANYTHING THE WIKI DOES NOT COVER
# * IS ASSUMED TO WORK THE SAME AS PrepareGetObject

async def get_object_infos(pid: int, data_ids: list[int]):
	"""
	Asks where to download several objects from as the given user,
	returning the infos and results
	"""

	async with helpers.connect(pid) as client:
		return await client.get_object_infos(data_ids)

async def get_object_infos_error(pid: int, data_ids: list[int]) -> str:
	"""
	Asks where to download several objects from and expects an error to be
	thrown for the whole request, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_object_infos(data_ids)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetObjectInfos did not throw an error")

async def get_one(pid: int, data_id: int) -> datastore.DataStoreReqGetInfo:
	"""
	Asks where to download one object from, expecting it to work, and
	returns its info
	"""

	response = await get_object_infos(pid, [data_id])

	assert helpers.result_names(response.results) == [None]

	return response.infos[0]

async def get_one_error(pid: int, data_id: int) -> str:
	"""
	Asks where to download one object from and expects it to fail,
	returning the name of its result
	"""

	response = await get_object_infos(pid, [data_id])
	[result] = helpers.result_names(response.results)

	if result is None:
		pytest.fail("DataStoreProtocol::GetObjectInfos did not fail the entry")

	return result

# * What the server hands back

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_returns_somewhere_to_download_from(unique_data_type, post):
	"""
	Basic functionality test
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	# TODO - The stock servers also send some headers and such here, but since those are implementation details we don't check for them
	assert (await get_one(nex.FRIEND_A, data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_size_is_the_one_the_object_was_posted_with(unique_data_type, post):
	"""
	The size comes from the object, not from whatever was uploaded
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, size=1234)

	assert (await get_one(nex.FRIEND_A, data_id)).size == 1234

@pytest.mark.skipif(nex.SERVER_NEX_VERSION < 30500, reason="dataId was added to DataStoreReqGetInfo in NEX 3.5")
async def test_the_data_id_is_the_objects(unique_data_type):
	"""
	Each info says which object it is for
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert (await get_one(nex.FRIEND_A, data_id)).data_id == data_id

async def test_the_entries_are_in_the_order_asked_for(unique_data_type):
	"""
	There is one info and one result for each data ID, in the same order
	"""

	first = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=10)
	second = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, size=20)
	response = await get_object_infos(nex.FRIEND_A, [second, first])

	assert helpers.result_names(response.results) == [None, None]
	assert [info.size for info in response.infos] == [20, 10]

async def test_a_data_id_asked_for_twice_comes_back_twice(unique_data_type):
	"""
	Each data ID gets its own entry, even when it is the same object
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_object_infos(nex.FRIEND_A, [data_id, data_id])

	assert helpers.result_names(response.results) == [None, None]
	assert len(response.infos) == 2

# * data_ids

async def test_no_data_ids():
	"""
	Asking for nothing gives nothing back
	"""

	response = await get_object_infos(nex.FRIEND_A, [])

	assert response.infos == []
	assert response.results == []

async def test_exactly_the_capacity(unique_data_type):
	"""
	BATCH_PROCESSING_CAPACITY data IDs are allowed
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_object_infos(nex.FRIEND_A, [data_id] * constants.BATCH_PROCESSING_CAPACITY)

	assert helpers.result_names(response.results) == [None] * constants.BATCH_PROCESSING_CAPACITY

async def test_too_many_data_ids(unique_data_type):
	"""
	More than BATCH_PROCESSING_CAPACITY data IDs fails the whole request
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await get_object_infos_error(nex.FRIEND_A, [data_id] * (constants.BATCH_PROCESSING_CAPACITY + 1)) == "DataStore::OverCapacity"

@pytest.mark.parametrize("data_id", [
	pytest.param(constants.INVALID_DATAID, id="zero"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(data_id):
	"""
	A data ID with no object behind it is missing
	"""

	assert await get_one_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_a_failure_only_fails_its_own_entry(unique_data_type):
	"""
	The objects which can be downloaded still get somewhere to download
	them from
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_object_infos(nex.FRIEND_A, [helpers.UNKNOWN_DATA_ID, data_id])

	assert helpers.result_names(response.results) == ["DataStore::NotFound", None]
	assert response.infos[1].url != ""

async def test_a_failed_entry_has_an_empty_info():
	"""
	Every data ID still gets an info, so one which failed gets an info
	which is zeroed
	"""

	[info] = (await get_object_infos(nex.FRIEND_A, [helpers.UNKNOWN_DATA_ID])).infos

	assert info.url == ""
	assert info.headers == []
	assert info.size == 0
	assert info.root_ca_cert == b""

	# * dataId was added to DataStoreReqGetInfo in NEX 3.5
	if nex.SERVER_NEX_VERSION >= 30500:
		assert info.data_id == constants.INVALID_DATAID

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	assert await get_one_error(nex.FRIEND_A, 999999) == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await get_one_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects have nothing to download
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await get_one_error(nex.FRIEND_A, data_id) == "DataStore::NotFound"

async def test_an_object_with_no_file_to_download(unique_data_type):
	"""
	An object posted with PostMetaBinary has no file on the file
	server, so there is nowhere to point the caller at
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)

	assert await get_one_error(nex.FRIEND_A, data_id) == "DataStore::OperationNotAllowed"

# * The access permission of the object

@pytest.mark.parametrize("permission, recipients, allowed, denied", helpers.ACCESS_PERMISSION_VIEWERS_AND_OUTSIDERS)
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_access_permission_of_the_object(unique_data_type, permission, recipients, allowed, denied, post):
	"""
	Downloading is gated the same way reading the metadata is
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=access)

	assert (await get_one(allowed, data_id)).url != ""

	if denied is not None:
		assert await get_one_error(denied, data_id) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("caller", [
	pytest.param(nex.FRIEND_B, id="a-friend"),
	pytest.param(nex.STRANGER, id="a-stranger")
])
@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_private_object_is_closed_to_everybody(unique_data_type, caller, post):
	"""
	A private object lets nobody in at all
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert (await get_one(nex.FRIEND_A, data_id)).url != ""
	assert await get_one_error(caller, data_id) == "DataStore::PermissionDenied"

# * The object status

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_pending_object_is_downloadable_by_its_owner(unique_data_type, post):
	"""
	An object under review can still be fetched by its owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert (await get_one(nex.FRIEND_A, data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_pending_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Everyone but the owner is told it is under review
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await get_one_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_rejected_object_is_downloadable_by_its_owner(unique_data_type, post):
	"""
	A rejected object can still be fetched by its owner
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert (await get_one(nex.FRIEND_A, data_id)).url != ""

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_rejected_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Being rejected is not told apart from being under review
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_one_error(nex.STRANGER, data_id) == "DataStore::UnderReviewing"

# * Asking where to download an object counts as referring to it

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_refers_to_it(unique_data_type, post):
	"""
	The reference count goes up and the referred time moves
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await get_one(nex.FRIEND_A, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count + 1
	assert after.referred_time.value() > before.referred_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_the_reference_count_stops_at_its_limit(unique_data_type, post):
	"""
	A count already at UINT32_MAX stays there rather than wrapping
	back around to 0
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=helpers.UINT32_MAX)
	await get_one(nex.FRIEND_A, data_id)

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).referred_count == helpers.UINT32_MAX

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_does_not_move_the_expiry(unique_data_type, post):
	"""
	An object without DATA_FLAG_PERIOD_FROM_LAST_REFERRED set does not
	have its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await get_one(nex.FRIEND_A, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() == before.expire_time.value()

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_getting_an_object_can_push_the_expiry_back(unique_data_type, post):
	"""
	An object with DATA_FLAG_PERIOD_FROM_LAST_REFERRED set has its expiry updated
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await get_one(nex.FRIEND_A, data_id)

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.expire_time.value() > before.expire_time.value() # TODO - More accurate check for this, right now this isn't super accurate

@pytest.mark.parametrize("post", helpers.FILE_SERVER_POST_METHODS)
async def test_a_failed_get_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Being refused the object does not count as having referred to it
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert await get_one_error(nex.STRANGER, data_id) == "DataStore::PermissionDenied"

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
