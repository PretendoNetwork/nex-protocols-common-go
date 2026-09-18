import contextlib

import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants

@contextlib.asynccontextmanager
async def connect(pid: int):
	"""
	Connects a user by PID to the test server, as a DataStore client
	"""

	async with nex.connect(pid) as client:
		yield datastore.DataStoreClient(client)

def permission(**fields) -> datastore.DataStorePermission:
	"""
	Creates a DataStorePermission with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePermission()
	param.permission = constants.Permission.PERMISSION_PUBLIC
	param.recipients = []

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def post_param(**fields) -> datastore.DataStorePreparePostParam:
	"""
	Creates a DataStorePreparePostParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePreparePostParam()
	param.size = 0
	param.name = "test"
	param.data_type = 0
	param.meta_binary = b"meta"
	param.permission = permission()
	param.delete_permission = permission(permission=constants.Permission.PERMISSION_PRIVATE)
	param.flag = constants.DataFlag.DATA_FLAG_NONE
	param.period = constants.DEFAULT_PERIOD
	param.extra_data = ["WUP", "4", "EUR", "110", "GB", ""]

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def post_param_v1(**fields) -> datastore.DataStorePreparePostParamV1:
	"""
	Creates a DataStorePreparePostParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePreparePostParamV1()
	param.size = 0
	param.name = "test"
	param.data_type = 0
	param.meta_binary = b"meta"
	param.permission = permission()
	param.delete_permission = permission(permission=constants.Permission.PERMISSION_PRIVATE)
	param.flag = constants.DataFlag.DATA_FLAG_NONE
	param.period = constants.DEFAULT_PERIOD
	param.refer_data_id = constants.INVALID_DATAID
	param.tags = []
	param.rating_init_param = []

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def delete_param(**fields) -> datastore.DataStoreDeleteParam:
	"""
	Creates a DataStoreDeleteParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreDeleteParam()
	param.data_id = constants.INVALID_DATAID
	param.update_password = constants.INVALID_PASSWORD

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def rating_init_param(**fields) -> datastore.DataStoreRatingInitParam:
	"""
	Creates a DataStoreRatingInitParam with default parameters, using
	no locks or limits. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreRatingInitParam()
	param.flag = 0
	param.internal_flag = 0
	param.lock_type = constants.RatingLockType.RATING_LOCK_NONE
	param.initial_value = 0
	param.range_min = 0
	param.range_max = 0
	param.period_hour = 0
	param.period_duration = 0

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def rating_init_param_with_slot(**fields) -> datastore.DataStoreRatingInitParamWithSlot:
	"""
	Creates a DataStoreRatingInitParamWithSlot with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreRatingInitParamWithSlot()
	param.slot = 0
	param.param = rating_init_param()

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def persistence_init_param(**fields) -> datastore.DataStorePersistenceInitParam:
	"""
	Creates a DataStorePersistenceInitParam with default parameters, which
	does not persist the object. Any field can be overridden with a keyword
	argument
	"""

	param = datastore.DataStorePersistenceInitParam()
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID
	param.delete_last_object = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def persistence_target(**fields) -> datastore.DataStorePersistenceTarget:
	"""
	Creates a DataStorePersistenceTarget with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStorePersistenceTarget()
	param.owner_id = 0
	param.persistence_id = constants.INVALID_PERSISTENCE_SLOT_ID

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def complete_post_param(**fields) -> datastore.DataStoreCompletePostParam:
	"""
	Creates a DataStoreCompletePostParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParam()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def change_meta_compare_param(**fields) -> datastore.DataStoreChangeMetaCompareParam:
	"""
	Creates a DataStoreChangeMetaCompareParam with default parameters,
	which compares nothing. Any field can be overridden with a keyword
	argument
	"""

	param = datastore.DataStoreChangeMetaCompareParam()
	param.comparison_flag = constants.ComparisonFlag.COMPARISON_FLAG_NONE
	param.name = ""
	param.permission = permission()
	param.delete_permission = permission()
	param.period = constants.DEFAULT_PERIOD
	param.meta_binary = b""
	param.tags = []
	param.referred_count = 0
	param.data_type = 0
	param.status = constants.DataStatus.DATA_STATUS_NONE

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def change_meta_param(**fields) -> datastore.DataStoreChangeMetaParam:
	"""
	Creates a DataStoreChangeMetaParam with default parameters, which
	changes nothing. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreChangeMetaParam()
	param.data_id = constants.INVALID_DATAID
	param.modifies_flag = constants.ModificationFlag.MODIFICATION_FLAG_NONE
	param.name = ""
	param.permission = permission()
	param.delete_permission = permission()
	param.period = constants.DEFAULT_PERIOD
	param.meta_binary = b""
	param.tags = []
	param.update_password = constants.INVALID_PASSWORD
	param.referred_count = 0
	param.data_type = 0
	param.status = constants.DataStatus.DATA_STATUS_NONE
	param.compare_param = change_meta_compare_param()
	param.persistence_target = persistence_target()

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def complete_post_param_v1(**fields) -> datastore.DataStoreCompletePostParamV1:
	"""
	Creates a DataStoreCompletePostParamV1 with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreCompletePostParamV1()
	param.data_id = constants.INVALID_DATAID
	param.success = True

	for name, value in fields.items():
		setattr(param, name, value)

	return param

def get_meta_param(**fields) -> datastore.DataStoreGetMetaParam:
	"""
	Creates a DataStoreGetMetaParam with default parameters.
	Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreGetMetaParam()
	param.data_id = constants.INVALID_DATAID
	param.persistence_target = persistence_target()
	param.result_option = 0
	param.access_password = 0

	for name, value in fields.items():
		setattr(param, name, value)

	return param

async def post_meta_binary(pid: int, **fields) -> int:
	"""
	Posts an object which does not use the file server as the
	given user, returning its data ID
	"""

	async with connect(pid) as client:
		return await client.post_meta_binary(post_param(**fields))

async def prepare_post_object(pid: int, **fields) -> int:
	"""
	Starts posting an object which uses the file server, as the given
	user, returning its data ID. The upload is not completed
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object(post_param(**fields))

		return post_info.data_id

async def prepare_post_object_v1(pid: int, **fields) -> int:
	"""
	Starts posting an object which uses the file server, as the given user,
	using the older version of the method. The upload is not completed
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object_v1(post_param_v1(**fields))

		return post_info.data_id

async def delete_object(pid: int, **fields) -> None:
	"""
	Deletes an object as the given user
	"""

	async with connect(pid) as client:
		await client.delete_object(delete_param(**fields))

async def post_object(pid: int, **fields) -> int:
	"""
	Posts an object which uses the file server as the given user, returning
	its data ID. The object contents is never uploaded
	"""

	async with connect(pid) as client:
		post_info = await client.prepare_post_object(post_param(**fields))
		await client.complete_post_object(complete_post_param(data_id=post_info.data_id))

		return post_info.data_id

async def change_meta(pid: int, **fields) -> None:
	"""
	Changes an objects metadata as the given user
	"""

	async with connect(pid) as client:
		await client.change_meta(change_meta_param(**fields))

async def get_password_info(pid: int, data_id: int) -> datastore.DataStorePasswordInfo:
	"""
	Gets an objects passwords as the given user. Only the object
	owner is allowed to do this
	"""

	async with connect(pid) as client:
		return await client.get_password_info(data_id)

async def get_meta(pid: int, **fields) -> datastore.DataStoreMetaInfo:
	"""
	Gets an objects metadata as the given user
	"""

	async with connect(pid) as client:
		return await client.get_meta(get_meta_param(**fields))

async def get_meta_error(pid: int, **fields) -> str:
	"""
	Runs DataStoreProtocol::GetMeta and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with connect(pid) as client:
		try:
			await client.get_meta(get_meta_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetMeta did not throw an error")

