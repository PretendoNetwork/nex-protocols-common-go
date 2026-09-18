import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def complete(pid: int, **fields) -> None:
	"""
	Completes an objects upload as the given user
	"""

	async with helpers.connect(pid) as client:
		await client.complete_post_object_v1(helpers.complete_post_param_v1(**fields))

async def complete_error(pid: int, **fields) -> str:
	"""
	Completes an objects upload and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.complete_post_object_v1(helpers.complete_post_param_v1(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::CompletePostObjectV1 did not throw an error")

@pytest.mark.parametrize("flag", [
	pytest.param(constants.DataFlag.DATA_FLAG_NONE, id="no-flags"),
	pytest.param(constants.DataFlag.DATA_FLAG_NEED_COMPLETION, id="need-completion"),
])
async def test_completes_an_upload(unique_data_type, flag):
	"""
	Objects which use the file server do not exist until their upload
	has been completed, whether or not DATA_FLAG_NEED_COMPLETION is set
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type, flag=flag)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

	await complete(nex.FRIEND_A, data_id=data_id)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id

async def test_completing_twice(unique_data_type):
	"""
	An objects upload can only be completed once
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)
	await complete(nex.FRIEND_A, data_id=data_id)

	assert await complete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::Unknown"

async def test_failed_upload(unique_data_type):
	"""
	Reporting a failed upload is not an error, but leaves the object
	as if it was never uploaded. The upload can still be completed later
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)
	await complete(nex.FRIEND_A, data_id=data_id, success=False)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

	await complete(nex.FRIEND_A, data_id=data_id)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id

async def test_reporting_a_failed_upload_twice(unique_data_type):
	"""
	Reporting a failed upload more than once is not an error, the
	object is simply left alone
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)

	await complete(nex.FRIEND_A, data_id=data_id, success=False)
	await complete(nex.FRIEND_A, data_id=data_id, success=False)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_reporting_a_failure_after_completing(unique_data_type):
	"""
	Reporting a failed upload after the upload was already completed is
	not an error, and does not hide the object again
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)

	await complete(nex.FRIEND_A, data_id=data_id)
	await complete(nex.FRIEND_A, data_id=data_id, success=False)

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(0xFFFFFF, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(0xFFFFFFFF, id="uint32-max"),
])
async def test_unknown_data_id(data_id):
	"""
	Completing an object which does not exist fails
	"""

	assert await complete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::Unknown"

async def test_only_the_owner_can_complete(unique_data_type):
	"""
	Only the object owner can complete its upload
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)

	assert await complete_error(nex.STRANGER, data_id=data_id) == "DataStore::PermissionDenied"

async def test_object_which_does_not_use_the_file_server(unique_data_type):
	"""
	Objects which do not use the file server have nothing to upload, so
	they are available right away and cannot be completed
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id
	assert await complete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::Unknown"

async def test_completing_a_deleted_object(unique_data_type):
	"""
	Deleted objects cannot be completed
	"""

	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type)

	await complete(nex.FRIEND_A, data_id=data_id)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await complete_error(nex.FRIEND_A, data_id=data_id) == "DataStore::Unknown"

async def test_completing_an_object_which_needs_review(unique_data_type):
	"""
	Completing an upload only marks the object as uploaded. An object
	which needs review is still under review afterwards
	"""

	flag = constants.DataFlag.DATA_FLAG_NEED_REVIEW
	data_id = await helpers.prepare_post_object_v1(nex.FRIEND_A, data_type=unique_data_type, flag=flag)

	await complete(nex.FRIEND_A, data_id=data_id)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.status == constants.DataStatus.DATA_STATUS_PENDING

async def test_completing_an_object_prepared_with_the_newer_version(unique_data_type):
	"""
	This method can complete an object which was started with
	PreparePostObject, the two versions can be mixed
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

	await complete(nex.FRIEND_A, data_id=data_id)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id
