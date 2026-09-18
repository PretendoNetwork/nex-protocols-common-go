import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

PERSISTENCE_SLOT = 0

# * Objects can be posted with or without the file server.
# * Testing both to ensure that GetMeta gets the same results
# * regardless of upload type
POST_METHODS = [
	pytest.param(helpers.post_meta_binary, id="post-meta-binary"),
	pytest.param(helpers.post_object, id="post-object"),
]

async def get_meta_error(pid: int, **fields) -> str:
	"""
	Runs DataStoreProtocol::GetMeta and expects an error to be thrown,
	returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_meta(helpers.get_meta_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetMeta did not throw an error")

async def post_persisted(post, pid: int, **fields) -> int:
	"""
	Posts an object as the given user, in the users persistence slot
	"""

	persistence_init_param = helpers.persistence_init_param(persistence_id=PERSISTENCE_SLOT)

	return await post(pid, persistence_init_param=persistence_init_param, **fields)

# * Access permission tests
# *
# * Objects are always visible to their owner. For everyone else, it
# * depends on the access permission of the object

ACCESS_PERMISSION_CASES = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.FRIEND_A, True, id="public-owner"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.FRIEND_B, True, id="public-friend"),
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, True, id="public-stranger"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_A, True, id="friend-owner"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, True, id="friend-friend"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.STRANGER, False, id="friend-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.FRIEND_A, True, id="specified-owner"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, True, id="specified-recipient"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.FRIEND_B, False, id="specified-not-recipient"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.FRIEND_A, True, id="private-owner"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.FRIEND_B, False, id="private-friend"),
	pytest.param(constants.Permission.PERMISSION_PRIVATE, [], nex.STRANGER, False, id="private-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.FRIEND_A, True, id="specified-friend-owner"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.FRIEND_B, True, id="specified-friend-recipient-friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B, nex.STRANGER], nex.STRANGER, False, id="specified-friend-recipient-stranger"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], nex.FRIEND_B, False, id="specified-friend-friend-not-recipient"),
]

@pytest.mark.parametrize("post", POST_METHODS)
@pytest.mark.parametrize("permission, recipients, viewer, visible", ACCESS_PERMISSION_CASES)
async def test_access_permission(unique_data_type, post, permission, recipients, viewer, visible):
	"""
	Tests general object visibility
	"""

	permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	if not visible:
		assert await get_meta_error(viewer, data_id=data_id) == "DataStore::PermissionDenied"
		return

	meta_info = await helpers.get_meta(viewer, data_id=data_id)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", POST_METHODS)
@pytest.mark.parametrize("permission, recipients, viewer, visible", ACCESS_PERMISSION_CASES)
async def test_access_permission_of_persisted_objects(unique_data_type, post, permission, recipients, viewer, visible):
	"""
	Tests persistence slot visibility
	"""

	permission = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)

	if not visible:
		assert await get_meta_error(viewer, persistence_target=target) == "DataStore::PermissionDenied"
		return

	meta_info = await helpers.get_meta(viewer, persistence_target=target)

	assert meta_info.data_id == data_id

# * Object status tests
# *
# * The object status is the objects visibility status, which is
# * separate from whether or not its upload was completed. Objects
# * which are under review cannot be seen by anyone besides the owner,
# * and objects which have been rejected cannot be seen by anyone
# * INCLUDING the owner, outside of SearchObject/SearchObjectLight

async def post_pending(post, pid: int, **fields) -> int:
	"""
	Posts an object which needs to be reviewed, as the given user
	"""

	return await post(pid, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW, **fields)

async def post_rejected(post, pid: int, **fields) -> int:
	"""
	Posts an object and marks it as rejected, as the given user
	"""

	data_id = await post(pid, **fields)

	await helpers.change_meta(pid, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=constants.DataStatus.DATA_STATUS_REJECTED)

	return data_id

@pytest.mark.parametrize("post", POST_METHODS)
async def test_pending_object_is_visible_to_its_owner(unique_data_type, post):
	"""
	Objects under review can still be seen by their owner
	"""

	data_id = await post_pending(post, nex.FRIEND_A, data_type=unique_data_type)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id
	assert meta_info.status == constants.DataStatus.DATA_STATUS_PENDING

@pytest.mark.parametrize("post", POST_METHODS)
async def test_pending_object_is_under_review_for_everyone_else(unique_data_type, post):
	"""
	Objects under review are not visible to anyone but their owner,
	even when the object is public
	"""

	data_id = await post_pending(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_meta_error(nex.STRANGER, data_id=data_id) == "DataStore::UnderReviewing"

@pytest.mark.parametrize("post", POST_METHODS)
async def test_rejected_object_is_not_visible_to_anyone(unique_data_type, post):
	"""
	Rejected objects behave as if they do not exist, for everyone
	including their owner
	"""

	data_id = await post_rejected(post, nex.FRIEND_A, data_type=unique_data_type)

	assert await get_meta_error(nex.STRANGER, data_id=data_id) == "DataStore::NotFound"
	assert await get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

# * Data ID lookup tests

@pytest.mark.parametrize("post", POST_METHODS)
async def test_data_id(unique_data_type, post):
	"""
	Tests that an object can be looked up by its dataId
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, name="object", period=30)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.data_id == data_id
	assert meta_info.owner_id == nex.FRIEND_A
	assert meta_info.name == "object"
	assert meta_info.data_type == unique_data_type
	assert meta_info.period == 30
	assert meta_info.status == constants.DataStatus.DATA_STATUS_NONE
	assert meta_info.permission.permission == constants.Permission.PERMISSION_PUBLIC
	assert meta_info.delete_permission.permission == constants.Permission.PERMISSION_PRIVATE

@pytest.mark.parametrize("post", POST_METHODS)
async def test_data_id_is_used_over_persistence_target(unique_data_type, post):
	"""
	If both a data ID and a persistence target are set, the data
	ID is used and the persistence target is ignored
	"""

	await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert meta_info.data_id == data_id

async def test_data_id_file_server_flag(unique_data_type):
	"""
	Tests whether or not the server returns DATA_FLAG_NOT_USE_FILESERVER
	on objects uploaded with PostMetaBinary
	"""

	# TODO - Is this really a "data ID" test, or more like a "flags" test?

	data_id = await helpers.post_object(nex.FRIEND_A, data_type=unique_data_type)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert not meta_info.flag & constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

	meta_binary_data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type)
	meta_binary_meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=meta_binary_data_id)

	assert meta_binary_meta_info.flag & constants.DataFlag.DATA_FLAG_NOT_USE_FILESERVER

async def test_unknown_data_id():
	"""
	Looking up an object which does not exist fails
	"""

	assert await get_meta_error(nex.FRIEND_A, data_id=0xFFFFFF) == "DataStore::NotFound"

async def test_no_data_id_or_persistence_target():
	"""
	Either a data ID or a persistence target must be set
	"""

	assert await get_meta_error(nex.FRIEND_A) == "DataStore::InvalidArgument"

# * Object persistence tests
# *
# * Somewhat redundant due to the access permission
# * tests, but cover all our bases anyway

@pytest.mark.parametrize("post", POST_METHODS)
async def test_persistence_target(unique_data_type, post):
	"""
	Tests that an object can be looked up by its persistence slot
	"""

	data_id = await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info = await helpers.get_meta(nex.FRIEND_A, persistence_target=target)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", POST_METHODS)
async def test_persistence_target_of_another_user(unique_data_type, post):
	"""
	Tests that another users persistence slot can be checked, assuming
	the caller has permission
	"""

	data_id = await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info = await helpers.get_meta(nex.STRANGER, persistence_target=target)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", POST_METHODS)
async def test_persistence_target_without_a_slot_is_ignored(unique_data_type, post):
	"""
	A persistence target which uses INVALID_PERSISTENCE_SLOT_ID is not
	used, even when it has an owner, and the data ID is used instead
	"""

	# TODO - Is this redundant because of test_data_id?

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	# * persistence_target defaults to using INVALID_PERSISTENCE_SLOT_ID
	target = helpers.persistence_target(owner_id=nex.FRIEND_A)
	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, persistence_target=target)

	assert meta_info.data_id == data_id

@pytest.mark.parametrize("post", POST_METHODS)
async def test_persistence_target_returns_the_newest_object(unique_data_type, post):
	"""
	Posting another object into a slot which is already in use
	replaces the object which was in it
	"""

	await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type)
	newest = await post_persisted(post, nex.FRIEND_A, data_type=unique_data_type)

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=PERSISTENCE_SLOT)
	meta_info = await helpers.get_meta(nex.FRIEND_A, persistence_target=target)

	assert meta_info.data_id == newest

async def test_empty_persistence_slot():
	"""
	Looking up a persistence slot which has no object in it fails
	"""

	target = helpers.persistence_target(owner_id=nex.FRIEND_A, persistence_id=constants.NUM_PERSISTENCE_SLOT - 1)

	assert await get_meta_error(nex.FRIEND_A, persistence_target=target) == "DataStore::NotFound"

# * ResultFlag tests
# *
# * ResultFlag determines what data is returned in the response

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_empty_by_default(unique_data_type, post):
	"""
	Optional metadata fields are left empty when no result flags are set
	"""

	rating_init_param = helpers.rating_init_param_with_slot()
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a"], rating_init_param=[rating_init_param])

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert meta_info.tags == []
	assert meta_info.meta_binary == b""
	assert meta_info.ratings == []
	assert meta_info.permission.recipients == []

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_tags(unique_data_type, post):
	"""
	RESULT_FLAG_TAGS returns the object tags
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a", "b"])

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_TAGS)

	assert meta_info.tags == ["a", "b"]

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_ratings(unique_data_type, post):
	"""
	RESULT_FLAG_RATINGS returns every rating slot of the object
	"""

	rating_init_param = helpers.rating_init_param_with_slot(param=helpers.rating_init_param(initial_value=7))
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, rating_init_param=[rating_init_param])

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_RATINGS)

	assert [(rating.slot, rating.info.total_value) for rating in meta_info.ratings] == [(0, 7)]

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_metabinary(unique_data_type, post):
	"""
	RESULT_FLAG_METABINARY returns the object MetaBinary
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, meta_binary=b"\x01\x02")

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_METABINARY)

	assert meta_info.meta_binary == b"\x01\x02"

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_permitted_ids(unique_data_type, post):
	"""
	RESULT_FLAG_PERMITTED_IDS returns the recipient ID lists
	from both the access and update permissions
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])
	delete_permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER])
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission, delete_permission=delete_permission)

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS)

	assert meta_info.permission.recipients == [nex.FRIEND_B]
	assert meta_info.delete_permission.recipients == [nex.STRANGER]

@pytest.mark.parametrize("post", POST_METHODS)
async def test_result_option_every_flag(unique_data_type, post):
	"""
	Every optional metadata field is returned when all result flags are set
	"""

	result_option = (
		constants.ResultFlag.RESULT_FLAG_TAGS |
		constants.ResultFlag.RESULT_FLAG_RATINGS |
		constants.ResultFlag.RESULT_FLAG_METABINARY |
		constants.ResultFlag.RESULT_FLAG_PERMITTED_IDS
	)
	permission = helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B])
	rating_init_param = helpers.rating_init_param_with_slot(param=helpers.rating_init_param(initial_value=7))
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, tags=["a"], meta_binary=b"\x01\x02", permission=permission, rating_init_param=[rating_init_param])

	meta_info = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=result_option)

	assert meta_info.tags == ["a"]
	assert meta_info.meta_binary == b"\x01\x02"
	assert meta_info.permission.recipients == [nex.FRIEND_B]
	assert [(rating.slot, rating.info.total_value) for rating in meta_info.ratings] == [(0, 7)]

# * Access password tests
# *
# * The server assigns every object a random access password, which the
# * owner can share with other users. A correct password bypasses every
# * permission check

@pytest.mark.parametrize("post", POST_METHODS)
async def test_no_access_password_without_permission(unique_data_type, post):
	"""
	Using INVALID_PASSWORD on an object the caller has no
	access to fails with PermissionDenied
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await get_meta_error(nex.STRANGER, data_id=data_id, access_password=constants.INVALID_PASSWORD) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", POST_METHODS)
async def test_wrong_access_password_without_permission(unique_data_type, post):
	"""
	Using the wrong password on an object the caller has no access
	to fails with InvalidPassword
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await get_meta_error(nex.STRANGER, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", POST_METHODS)
async def test_wrong_access_password_with_permission(unique_data_type, post):
	"""
	Using the wrong password fails even when the caller already
	has access to the object without one
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	assert await get_meta_error(nex.STRANGER, data_id=data_id, access_password=1234) == "DataStore::InvalidPassword"

@pytest.mark.parametrize("post", POST_METHODS)
async def test_access_password_bypasses_permissions(unique_data_type, post):
	"""
	Using the correct password bypasses every permission check
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	password_info = await helpers.get_password_info(nex.FRIEND_A, data_id)
	meta_info = await helpers.get_meta(nex.STRANGER, data_id=data_id, access_password=password_info.access_password)

	assert meta_info.data_id == data_id
