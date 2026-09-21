import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

async def get_password_infos(pid: int, data_ids: list[int]):
	"""
	Gets the passwords of several objects as the given user
	"""

	async with helpers.connect(pid) as client:
		return await client.get_password_infos(data_ids)

async def get_password_infos_error(pid: int, data_ids: list[int]) -> str:
	"""
	Gets the passwords of several objects and expects an error to be
	thrown, returning the name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.get_password_infos(data_ids)
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::GetPasswordInfos did not throw an error")

def data_ids_of(response) -> list[int]:
	"""
	The data ID of every info which came back
	"""

	return [info.data_id for info in response.infos]

# * The passwords the server made when the objects were posted

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_returns_the_passwords(unique_data_type, post):
	"""
	The owner of an object can read its passwords, and that
	the passwords differ from each other
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	response = await get_password_infos(nex.FRIEND_A, [data_id])
	info = response.infos[0]

	assert helpers.result_names(response.results) == [None]
	assert info.data_id == data_id
	assert info.access_password != constants.INVALID_PASSWORD
	assert info.update_password != constants.INVALID_PASSWORD
	assert info.access_password != info.update_password

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_passwords_do_not_change(unique_data_type, post):
	"""
	Reading the passwords twice gives the same answer
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	first = (await get_password_infos(nex.FRIEND_A, [data_id])).infos[0]
	second = (await get_password_infos(nex.FRIEND_A, [data_id])).infos[0]

	assert first.access_password == second.access_password
	assert first.update_password == second.update_password

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_every_object_has_its_own_passwords(unique_data_type, post):
	"""
	Two objects posted the same way do not share passwords
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	response = await get_password_infos(nex.FRIEND_A, [first, second])

	assert response.infos[0].access_password != response.infos[1].access_password
	assert response.infos[0].update_password != response.infos[1].update_password

# * Password entropy tests. It is not possible to test if a password
# * is cryptographically secure without triggering false negatives,
# * but we can at least catch obviously bad passwords

UINT64_MAX = 0xFFFFFFFFFFFFFFFF
UINT32_MAX = 0xFFFFFFFF

# * How many objects to post when looking at entropy
PASSWORD_SAMPLES = 16

# * The smallest gap allowed between two sorted passwords. A good
# * generator leaves huge gaps between passwords. This tries to
# * catch sequential generators and generators which produce small
# * gaps, but is intentionally somewhat small as to not trip any
# * false negatives
CLOSEST_SENSIBLE_PASSWORDS = 1000000

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_passwords_are_not_small(unique_data_type, post):
	"""
	A password fills the whole 64 bit range, so one which fits in 32
	bits likely comes from an insecure counter/generator
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	info = (await get_password_infos(nex.FRIEND_A, [data_id])).infos[0]

	# * In my live tests the smallest password was 60 bits, so this
	# * gives a lot more leway for potentially bad passwords

	assert info.access_password > UINT32_MAX
	assert info.update_password > UINT32_MAX

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_passwords_are_spread_out(unique_data_type, post):
	"""
	Tests that passwords have sufficient gaps/bit usage in their data
	"""

	data_ids = [await post(nex.FRIEND_A, data_type=unique_data_type) for _ in range(PASSWORD_SAMPLES)]
	response = await get_password_infos(nex.FRIEND_A, data_ids)
	passwords = []

	for info in response.infos:
		passwords += [info.access_password, info.update_password]

	assert len(set(passwords)) == len(passwords)

	ordered = sorted(passwords)

	# * Make sure passwords, when sorted, have nice big gaps, to try and catch
	# * sequential generators
	assert min(b - a for a, b in zip(ordered, ordered[1:])) > CLOSEST_SENSIBLE_PASSWORDS

	# * Ensure an even spread of bits across all password samples. Ideally
	# * the password generator should be generating passwords with enough
	# * randomness that every bit in the 64 bit space gets set/unset across
	# * all samples. If there's ever a bit that is not used, that's likely
	# * a sign of a bad generator
	seen_set = 0
	seen_clear = 0

	for password in passwords:
		seen_set |= password
		seen_clear |= password ^ UINT64_MAX

	assert seen_set == UINT64_MAX
	assert seen_clear == UINT64_MAX

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_update_password_works(unique_data_type, post):
	"""
	The password which comes back actually works
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	info = (await get_password_infos(nex.FRIEND_A, [data_id])).infos[0]

	await helpers.delete_object(nex.STRANGER, data_id=data_id, update_password=info.update_password)

# * Only the owner may read an objects passwords. The access permission
# * is a separate thing which says who may read the object itself

ACCESS_PERMISSIONS = [
	pytest.param(constants.Permission.PERMISSION_PUBLIC, [], nex.STRANGER, id="public"),
	pytest.param(constants.Permission.PERMISSION_FRIEND, [], nex.FRIEND_B, id="friend"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], nex.STRANGER, id="specified"),
	pytest.param(constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], nex.FRIEND_B, id="specified-friend"),
]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("permission, recipients, caller", ACCESS_PERMISSIONS)
async def test_the_access_permission_does_not_open_the_passwords(unique_data_type, post, permission, recipients, caller):
	"""
	Whoever the access permission lets in can read the object itself,
	but still cannot read its passwords
	"""

	access = helpers.permission(permission=permission, recipients=recipients)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=access)

	assert (await helpers.get_meta(caller, data_id=data_id)).data_id == data_id

	response = await get_password_infos(caller, [data_id])

	assert helpers.result_names(response.results) == ["DataStore::PermissionDenied"]
	assert data_ids_of(response) == [constants.INVALID_DATAID]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("caller", [
	pytest.param(nex.FRIEND_B, id="a-friend"),
	pytest.param(nex.STRANGER, id="a-stranger"),
])
async def test_nobody_else_can_read_a_private_objects_passwords(unique_data_type, post, caller):
	"""
	A private object lets nobody in at all. Tested separately because we
	expect the GetMeta call to fail
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)

	assert await helpers.get_meta_error(caller, data_id=data_id) == "DataStore::PermissionDenied"

	response = await get_password_infos(caller, [data_id])

	assert helpers.result_names(response.results) == ["DataStore::PermissionDenied"]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_somebody_elses_object_only_fails_its_own_entry(unique_data_type, post):
	"""
	An object the caller does not own fails on its own, leaving the
	rest of the list to go through
	"""

	mine = await post(nex.FRIEND_A, data_type=unique_data_type)
	theirs = await post(nex.STRANGER, data_type=unique_data_type)

	response = await get_password_infos(nex.FRIEND_A, [mine, theirs])

	assert helpers.result_names(response.results) == [None, "DataStore::PermissionDenied"]
	assert data_ids_of(response) == [mine, constants.INVALID_DATAID]

# * Objects the owner can still read the passwords of

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_an_object_under_review(unique_data_type, post):
	"""
	An object which is under review is hidden from everybody else,
	but its owner can still read its passwords
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)
	response = await get_password_infos(nex.FRIEND_A, [data_id])

	assert helpers.result_names(response.results) == [None]
	assert data_ids_of(response) == [data_id]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_owner_of_a_private_object(unique_data_type, post):
	"""
	The access permission of the object makes no difference to its
	owner reading its passwords
	"""

	permission = helpers.permission(permission=constants.Permission.PERMISSION_PRIVATE)
	data_id = await post(nex.FRIEND_A, data_type=unique_data_type, permission=permission)
	response = await get_password_infos(nex.FRIEND_A, [data_id])

	assert helpers.result_names(response.results) == [None]
	assert data_ids_of(response) == [data_id]

# * Objects which are not there

@pytest.mark.parametrize("data_id", [
	pytest.param(helpers.UNKNOWN_DATA_ID, id="unknown"),
	pytest.param(constants.INVALID_DATAID, id="invalid-dataid"),
	pytest.param(999999, id="the-system-range"),
	pytest.param(UINT64_MAX, id="uint64-max"),
])
async def test_unknown_data_id(data_id):
	"""
	Reading the passwords of an object which does not exist fails
	"""

	response = await get_password_infos(nex.FRIEND_A, [data_id])

	assert helpers.result_names(response.results) == ["DataStore::NotFound"]
	assert data_ids_of(response) == [constants.INVALID_DATAID]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_a_deleted_object(unique_data_type, post):
	"""
	Deleted objects have no passwords to read
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	response = await get_password_infos(nex.FRIEND_A, [data_id])

	assert helpers.result_names(response.results) == ["DataStore::NotFound"]

async def test_an_object_which_was_never_completed(unique_data_type):
	"""
	An object whose upload was never completed does not exist yet
	"""

	data_id = await helpers.prepare_post_object(nex.FRIEND_A, data_type=unique_data_type)

	response = await get_password_infos(nex.FRIEND_A, [data_id])

	assert helpers.result_names(response.results) == ["DataStore::NotFound"]

# * The response is two lists which line up with the data IDs which were
# * asked for, one of infos and one of results

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_returns_an_info_for_every_object(unique_data_type, post):
	"""
	Every object in the list gets its own passwords back, in the order
	they were asked for
	"""

	first = await post(nex.FRIEND_A, data_type=unique_data_type)
	second = await post(nex.FRIEND_A, data_type=unique_data_type)

	response = await get_password_infos(nex.FRIEND_A, [second, first])

	assert helpers.result_names(response.results) == [None, None]
	assert data_ids_of(response) == [second, first]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
@pytest.mark.parametrize("position", [
	pytest.param(0, id="the-failure-comes-first"),
	pytest.param(1, id="the-failure-comes-last"),
])
async def test_a_failure_does_not_fail_the_request(unique_data_type, post, position):
	"""
	An object which is not there fails on its own
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)
	data_ids = [helpers.UNKNOWN_DATA_ID, data_id] if position == 0 else [data_id, helpers.UNKNOWN_DATA_ID]

	response = await get_password_infos(nex.FRIEND_A, data_ids)

	if position == 0:
		assert helpers.result_names(response.results) == ["DataStore::NotFound", None]
		assert data_ids_of(response) == [constants.INVALID_DATAID, data_id]
	else:
		assert helpers.result_names(response.results) == [None, "DataStore::NotFound"]
		assert data_ids_of(response) == [data_id, constants.INVALID_DATAID]

@pytest.mark.parametrize("post", helpers.POST_METHODS)
async def test_the_same_object_twice(unique_data_type, post):
	"""
	Reading the same object multiple times is supported
	"""

	data_id = await post(nex.FRIEND_A, data_type=unique_data_type)

	response = await get_password_infos(nex.FRIEND_A, [data_id, data_id])

	assert helpers.result_names(response.results) == [None, None]
	assert data_ids_of(response) == [data_id, data_id]
	assert response.infos[0].access_password == response.infos[1].access_password

async def test_no_data_ids():
	"""
	Asking for nothing at all returns nothing
	"""

	response = await get_password_infos(nex.FRIEND_A, [])

	assert response.infos == []
	assert response.results == []

async def test_exactly_the_capacity():
	"""
	The capacity itself is allowed
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * constants.BATCH_PROCESSING_CAPACITY
	response = await get_password_infos(nex.FRIEND_A, data_ids)

	assert helpers.result_names(response.results) == ["DataStore::NotFound"] * len(data_ids)

async def test_too_many_data_ids():
	"""
	One object over the capacity is refused outright
	"""

	data_ids = [helpers.UNKNOWN_DATA_ID] * (constants.BATCH_PROCESSING_CAPACITY + 1)

	assert await get_password_infos_error(nex.FRIEND_A, data_ids) == "DataStore::OverCapacity"
