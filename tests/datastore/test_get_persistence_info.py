import pytest

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

SLOT = 0
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

# * What the slot holds

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_returns_the_object_in_the_slot(unique_data_type, post):
	"""
	The slot holds the data ID of the object which is in it
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type)
	info = await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT)

	assert info.owner_id == nex.FRIEND_A
	assert info.slot_id == SLOT
	assert info.data_id == data_id

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_the_newest_object_wins(unique_data_type, post):
	"""
	Putting another object in a slot replaces the one which was there
	"""

	await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type)
	second = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT)).data_id == second

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_deleting_the_object_empties_the_slot(unique_data_type, post):
	"""
	Deleting a persisted object leaves the slot with nothing in it
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, SLOT) == "DataStore::NotFound"

# * Who may read somebody elses slot

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
@pytest.mark.parametrize("permission, recipients, allowed, denied", ACCESS_PERMISSIONS)
async def test_the_access_permission_of_the_object(unique_data_type, post, permission, recipients, allowed, denied):
	"""
	Reading a slot is gated by the access permission of the object in it
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type, permission=access)

	assert (await helpers.get_persistence_info(allowed, nex.FRIEND_A, SLOT)).data_id == data_id

	if denied is not None:
		assert await helpers.get_persistence_info_error(denied, nex.FRIEND_A, SLOT) == "DataStore::PermissionDenied"

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
	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type, permission=permission)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT)).data_id == data_id
	assert await helpers.get_persistence_info_error(caller, nex.FRIEND_A, SLOT) == "DataStore::PermissionDenied"

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_object_under_review(unique_data_type, post):
	"""
	An object waiting to be reviewed reports as such, but is viewable to
	its owner like normal
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert (await helpers.get_persistence_info(nex.FRIEND_A, nex.FRIEND_A, SLOT)).data_id == data_id
	assert await helpers.get_persistence_info_error(nex.STRANGER, nex.FRIEND_A, SLOT) == "DataStore::UnderReviewing"

# * The owner being asked about

@pytest.mark.parametrize("post", helpers.PERSISTENCE_POST_METHODS)
async def test_an_owner_of_zero_means_the_caller(unique_data_type, post):
	"""
	Asking about nobody in particular reads the callers own slots
	"""

	data_id = await helpers.post_persisted(post, nex.FRIEND_A, SLOT, data_type=unique_data_type)
	info = await helpers.get_persistence_info(nex.FRIEND_A, 0, SLOT)

	assert info.data_id == data_id

async def test_unknown_owner():
	"""
	A user who has never persisted anything has no slots to read
	"""

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, 1234, SLOT) == "DataStore::NotFound"

# * Slots which hold nothing, or are not slots at all

async def test_empty_slot():
	"""
	A slot with no object in it has nothing to return
	"""

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, EMPTY_SLOT) == "DataStore::NotFound"

@pytest.mark.parametrize("slot", [
	pytest.param(constants.NUM_PERSISTENCE_SLOT, id="out-of-range-slot"),
	pytest.param(constants.INVALID_PERSISTENCE_SLOT_ID, id="invalid-slot")
])
async def test_invalid_slot(slot):
	"""
	Only slots 0 to NUM_PERSISTENCE_SLOT - 1 can be used
	"""

	assert await helpers.get_persistence_info_error(nex.FRIEND_A, nex.FRIEND_A, slot) == "DataStore::InvalidArgument"
