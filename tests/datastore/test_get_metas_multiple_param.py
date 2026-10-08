import anyio
import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

PERSISTENCE_SLOT = 0

# * Long enough for a timestamp to visibly move
TIMESTAMP_DELAY = 2

async def get_metas(pid: int, params: list[dict]):
	"""
	Gets the metadata of several objects as the given user. Each entry
	is the fields of one param
	"""

	async with helpers.connect(pid) as client:
		return await client.get_metas_multiple_param([helpers.get_meta_param(**fields) for fields in params])

async def get_metas_error(pid: int, params: list[dict]) -> str:
	"""
	Gets the metadata of several objects and expects an error to be
	thrown for the whole request, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_metas_multiple_param([helpers.get_meta_param(**fields) for fields in params])
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetMetasMultipleParam did not throw an error")

async def get_one(pid: int, **fields):
	"""
	Gets the metadata of a single object, returning its entry and the
	result for it
	"""

	response = await get_metas(pid, [fields])

	return response.infos[0], helpers.result_names(response.results)[0]

# * params

async def test_the_entries_are_in_the_order_asked_for(unique_data_type):
	"""
	Every param gets an entry, in the order the params were given
	"""

	data_ids = [await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type) for _ in range(3)]
	response = await get_metas(nex.FRIEND_A, [{"data_id": data_id} for data_id in reversed(data_ids)])

	assert helpers.result_names(response.results) == [None, None, None]
	assert [meta_info.data_id for meta_info in response.infos] == list(reversed(data_ids))

async def test_the_same_param_twice(unique_data_type):
	"""
	Each param gets its own entry, even when it is the same object
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_metas(nex.FRIEND_A, [
		{
			"data_id": data_id
		},
		{
			"data_id": data_id
		}
	])

	assert [meta_info.data_id for meta_info in response.infos] == [data_id, data_id]

async def test_no_params():
	"""
	Asking for nothing is not an error
	"""

	response = await get_metas(nex.FRIEND_A, [])

	assert response.infos == []
	assert response.results == []

async def test_exactly_the_capacity(unique_data_type):
	"""
	BATCH_PROCESSING_CAPACITY params are allowed
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_metas(nex.FRIEND_A, [{"data_id": data_id}] * constants.BATCH_PROCESSING_CAPACITY)

	assert len(response.infos) == constants.BATCH_PROCESSING_CAPACITY

async def test_too_many_params(unique_data_type):
	"""
	More than BATCH_PROCESSING_CAPACITY params fails the whole request
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)

	assert await get_metas_error(nex.FRIEND_A, [{"data_id": data_id}] * (constants.BATCH_PROCESSING_CAPACITY + 1)) == "DataStore::OverCapacity"

# * data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_data_id(unique_data_type, post):
	"""
	Tests that an object can be looked up by its dataId
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, name="object", period=30)
	meta_info, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result is None
	assert meta_info.data_id == data_id
	assert meta_info.owner_id == nex.FRIEND_A
	assert meta_info.name == "object"
	assert meta_info.data_type == unique_data_type
	assert meta_info.period == 30
	assert meta_info.status == constants.DataStatus.DATA_STATUS_NONE
	assert meta_info.permission.permission == constants.Permission.PERMISSION_PUBLIC
	assert meta_info.delete_permission.permission == constants.Permission.PERMISSION_PRIVATE

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_target_is_used_over_data_id(unique_data_type, post):
	"""
	If both a persistence target and a data ID are set, the
	persistence target is used and the data ID is ignored
	"""

	persisted = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	other = await post(nex.FRIEND_A, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=other, persistence_target=target)

	assert meta_info.data_id == persisted

async def test_data_id_file_server_flag(unique_data_type):
	"""
	Objects uploaded with PostMetaBinary have DATA_FLAG_NOT_USE_FILESERVER,
	objects on the file server do not
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	meta_binary_data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_metas(nex.FRIEND_A, [
		{
			"data_id": data_id
		},
		{
			"data_id": meta_binary_data_id
		}
	])

	assert not response.infos[0].flag & constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER
	assert response.infos[1].flag & constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

@pytest.mark.parametrize("missing", [
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown")
])
async def test_a_data_id_which_names_nothing(unique_data_type, missing):
	"""
	An object which cannot be found only fails its own entry, which is
	left empty. The rest still come back
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_metas(nex.FRIEND_A, [
		{
			"data_id": data_id
		},
		{
			"data_id": missing,
			"result_option": constants.ResultFlag.RESULT_FLAG_ALL
		}
	])

	assert helpers.result_names(response.results) == [None, "DataStore::NotFound"]
	assert response.infos[0].data_id == data_id
	assert response.infos[1] == helpers.empty_meta_info()

async def test_a_system_range_data_id():
	"""
	Data IDs below 1,000,000 belong to the system. Normal clients
	shouldn't interact with these objects. See
	https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	for details
	"""

	# TODO - This only returns NotFound because there's no way for a normal client to upload in this range, so there are no objects in the test server
	_, result = await get_one(nex.FRIEND_A, data_id=999999)

	assert result == "DataStore::NotFound"

async def test_a_deleted_object(unique_data_type):
	"""
	Deleted objects cannot be found
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	_, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result == "DataStore::NotFound"

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	Objects whose upload was never completed do not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)
	_, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result == "DataStore::NotFound"

async def test_an_expired_object(unique_data_type):
	"""
	An object with a period of 0 has already expired
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_PERIOD, period=0)

	_, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result == "DataStore::NotFound"

# * persistence_target

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_target(unique_data_type, post):
	"""
	Tests that an object can be looked up by its persistence slot
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info, result = await get_one(nex.FRIEND_A, persistence_target=target)

	assert result is None
	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_a_persistence_target_only_applies_to_its_own_entry(unique_data_type, post):
	"""
	Every param has its own persistence target, so the other entries are
	still looked up by their own data ID
	"""

	persisted = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	response = await get_metas(nex.FRIEND_A, [
		{
			"persistence_target": target
		},
		{
			"data_id": data_id
		}
	])

	assert [meta_info.data_id for meta_info in response.infos] == [persisted, data_id]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_target_of_another_user(unique_data_type, post):
	"""
	Tests that another users persistence slot can be checked, assuming
	the caller has permission
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info, _ = await get_one(nex.STRANGER, persistence_target=target)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_persistence_target_without_a_slot_is_ignored(unique_data_type, post):
	"""
	A persistence target which uses INVALID_PERSISTENCE_SLOT_ID is not
	used, even when it has an owner, and the data ID is used instead
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	# * persistence_target defaults to using INVALID_PERSISTENCE_SLOT_ID
	target = helpers.persistence_target(owner_id=nex.FRIEND_A)
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_target_returns_the_newest_object(unique_data_type, post):
	"""
	Posting another object into a slot which is already in use
	replaces the object which was in it
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	newest = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info, _ = await get_one(nex.FRIEND_A, persistence_target=target)

	assert meta_info.data_id == newest

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_slots_are_per_user(unique_data_type, post):
	"""
	Persistence slots are tied to a user, so the same slot ID
	holds a different object for every user
	"""

	first = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	second = await helpers.post_persisted(post, nex.FRIEND_B, PERSISTENCE_SLOT, data_type=unique_data_type)

	first_target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	second_target = helpers.persistence_target(owner_id=nex.FRIEND_B, persistence_id=PERSISTENCE_SLOT)
	response = await get_metas(nex.FRIEND_A, [
		{
			"persistence_target": first_target
		},
		{
			"persistence_target": second_target
		}
	])

	assert [meta_info.data_id for meta_info in response.infos] == [first, second]

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_persistence_target_without_an_owner(unique_data_type, post):
	"""
	A persistence target with a slot but no owner is still used, and
	fails its entry because slots are tied to a user
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type)
	target = helpers.persistence_target(owner_id=0, persistence_id=PERSISTENCE_SLOT)
	_, result = await get_one(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert result == "DataStore::NotFound"

async def test_empty_persistence_slot():
	"""
	Looking up a persistence slot which has no object in it fails
	its entry
	"""

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.NUM_PERSISTENCE_SLOT - 1)
	_, result = await get_one(nex.FRIEND_A, persistence_target=target)

	assert result == "DataStore::NotFound"

# * result_option

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_empty_by_default(unique_data_type, post):
	"""
	Optional metadata fields are left empty when no result flags are set
	"""

	rating_init_param = helpers.rating_init_param_with_slot()
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a"], rating_init_param=[rating_init_param])
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id)

	assert meta_info.tags == []
	assert meta_info.meta_binary == b""
	assert meta_info.ratings == []
	assert meta_info.permission.recipients == []

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_tags(unique_data_type, post):
	"""
	RESULT_FLAG_TAGS returns the object tags
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a", "b"])
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert meta_info.tags == ["a", "b"]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_ratings(unique_data_type, post):
	"""
	RESULT_FLAG_RATINGS returns every rating slot of the object
	"""

	rating_init_param = helpers.rating_init_param_with_slot(param=helpers.rating_init_param(initial_value=7))
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[rating_init_param])
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)

	assert [(rating.slot, rating.info.total_value) for rating in meta_info.ratings] == [(0, 7)]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_metabinary(unique_data_type, post):
	"""
	RESULT_FLAG_METABINARY returns the object MetaBinary
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"\x01\x02")
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_METABINARY)

	assert meta_info.meta_binary == b"\x01\x02"

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_permitted_ids(unique_data_type, post):
	"""
	RESULT_FLAG_PERMITTED_IDS returns the recipient ID lists
	from both the access and update permissions
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])
	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission, delete_permission=delete_permission)
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert meta_info.permission.recipients == [nex.FRIEND_B]
	assert meta_info.delete_permission.recipients == [nex.STRANGER]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_result_option_every_flag(unique_data_type, post):
	"""
	Every optional metadata field is returned when all result flags are set
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])
	rating_init_param = helpers.rating_init_param_with_slot(param=helpers.rating_init_param(initial_value=7))
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a"], meta_binary=b"\x01\x02", permission=permission, rating_init_param=[rating_init_param])
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert meta_info.tags == ["a"]
	assert meta_info.meta_binary == b"\x01\x02"
	assert meta_info.permission.recipients == [nex.FRIEND_B]
	assert [(rating.slot, rating.info.total_value) for rating in meta_info.ratings] == [(0, 7)]

async def test_each_entry_has_its_own_result_option(unique_data_type):
	"""
	The same object can be asked for with different result flags, and
	each entry only has what its own param asked for
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, tags=["a"], meta_binary=b"\x01\x02")
	response = await get_metas(nex.FRIEND_A, [
		{
			"data_id": data_id
		},
		{
			"data_id": data_id,
			"result_option": constants.ResultFlag.RESULT_FLAG_TAGS
		},
		{
			"data_id": data_id,
			"result_option": constants.ResultFlag.RESULT_FLAG_METABINARY
		}
	])

	assert [(meta_info.tags, meta_info.meta_binary) for meta_info in response.infos] == [([], b""), (["a"], b""), ([], b"\x01\x02")]

# * access_password

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_no_access_password_without_permission(unique_data_type, post):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails its entry with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	meta_info, result = await get_one(nex.STRANGER, data_id=data_id, access_password=constants.INVALID_PASSWORD, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert result == "DataStore::PermissionDenied"
	assert meta_info == helpers.empty_meta_info()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_access_password_without_permission(unique_data_type, post):
	"""
	Using the wrong password on an object the caller has no access
	to fails its entry with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	meta_info, result = await get_one(nex.STRANGER, data_id=data_id, access_password=1234, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert result == "DataStore::InvalidPassword"
	assert meta_info == helpers.empty_meta_info()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_wrong_access_password_with_permission(unique_data_type, post):
	"""
	Using the wrong password on an object the user already access to
	should not throw any errors
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	meta_info, result = await get_one(nex.STRANGER, data_id=data_id, access_password=1234)

	assert result is None
	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_access_password_bypasses_permissions(unique_data_type, post):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)
	meta_info, result = await get_one(nex.STRANGER, data_id=data_id, access_password=password_info.access_password)

	assert result is None
	assert meta_info.data_id == data_id

async def test_each_entry_has_its_own_access_password(unique_data_type):
	"""
	Every param carries the password for its own object, so several
	objects the caller has no access to can be opened at once
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	first = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	second = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	first_password_info = await helpers.get_password_info(nex.FRIEND_A, first)
	second_password_info = await helpers.get_password_info(nex.FRIEND_A, second)
	response = await get_metas(nex.STRANGER, [
		{
			"data_id": first,
			"access_password": first_password_info.access_password
		},
		{
			"data_id": second,
			"access_password": second_password_info.access_password
		}
	])

	assert helpers.result_names(response.results) == [None, None]
	assert [meta_info.data_id for meta_info in response.infos] == [first, second]

# * Access permission tests
# *
# * Objects are always visible to their owner. For everyone else, it
# * depends on the access permission of the object

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, post, permission, recipients, viewer, visible):
	"""
	Tests general object visibility. An object the viewer cannot see
	only fails its own entry
	"""

	permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	meta_info, result = await get_one(viewer, data_id=data_id)

	if not visible:
		assert result == "DataStore::PermissionDenied"
		return

	assert result is None
	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
@pytest.mark.parametrize("permission, recipients, viewer, visible", helpers.ACCESS_PERMISSION_CASES)
async def test_access_permission_of_persisted_objects(unique_data_type, post, permission, recipients, viewer, visible):
	"""
	Tests persistence slot visibility
	"""

	permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_persisted(post, nex.FRIEND_A, PERSISTENCE_SLOT, data_type=unique_data_type, permission=permission)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info, result = await get_one(viewer, persistence_target=target)

	if not visible:
		assert result == "DataStore::PermissionDenied"
		return

	assert result is None
	assert meta_info.data_id == data_id

# * Object status tests
# *
# * The object status is the objects visibility status, which is
# * separate from whether or not its upload was completed
# TODO - Previous tests before the NN shutdown indicated that DATA_STATUS_REJECTED resulted in NotFound
#        for those objects, for everyone, including the owner, unless SearchObject was used. However tests
#        on Pokemon Bank after the NN shutdown now contradict that. These tests are designed to match
#        Pokemon Bank until more information is confirmed

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_pending_object_is_visible_to_its_owner(unique_data_type, post):
	"""
	Objects under review can still be seen by their owner
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	meta_info, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result is None
	assert meta_info.status == constants.DataStatus.DATA_STATUS_PENDING

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_pending_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Objects under review are not visible to anyone but their owner,
	even when the object is public
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	meta_info, result = await get_one(nex.STRANGER, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)

	assert result == "DataStore::UnderReviewing"
	assert meta_info == helpers.empty_meta_info()

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_rejected_object_is_visible_to_its_owner(unique_data_type, post):
	"""
	Rejected objects can still be seen by their owner
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)
	meta_info, result = await get_one(nex.FRIEND_A, data_id=data_id)

	assert result is None
	assert meta_info.status == constants.DataStatus.DATA_STATUS_REJECTED

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_rejected_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Rejected objects appear as under review still to other users
	"""

	data_id = await helpers.post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)
	_, result = await get_one(nex.STRANGER, data_id=data_id)

	assert result == "DataStore::UnderReviewing"

# * Read lock tests
# *
# * Read locks only gate downloading an object, so they never stop
# * anyone from reading its metadata

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_read_locks_do_not_hide_metadata(unique_data_type, post):
	"""
	An object which is locked by someone else can still have its
	metadata read, by anyone who has access to it
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)

	# * Touching the object with a lock ID locks it
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id, lock_id=1234)

	assert (await get_one(nex.FRIEND_A, data_id=data_id))[0].data_id == data_id
	assert (await get_one(nex.STRANGER, data_id=data_id))[0].data_id == data_id

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_read_lock_flag_is_returned(unique_data_type, post):
	"""
	Objects posted with DATA_FLAG_USE_READ_LOCK say so in their metadata
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_USE_READ_LOCK)
	meta_info, _ = await get_one(nex.FRIEND_A, data_id=data_id)

	assert meta_info.flag & constants.DataFlag.DATA_FLAG_USE_READ_LOCK

# * Reading the metadata does not count as referring to the object

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_reading_the_metadata_does_not_refer_to_the_object(unique_data_type, post):
	"""
	Only downloading or touching an object refers to it, so neither the
	reference count nor the referred time move, even when everything
	about the object is asked for
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_PERIOD_FROM_LAST_REFERRED)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(TIMESTAMP_DELAY)
	await get_metas(nex.STRANGER, [
		{
			"data_id": data_id,
			"result_option": constants.ResultFlag.RESULT_FLAG_ALL
		}
	])

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time.value() == before.referred_time.value()
	assert after.expire_time.value() == before.expire_time.value()
