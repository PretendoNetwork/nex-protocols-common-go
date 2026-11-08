import uuid

import anyio
import pytest
from nintendo.nex import common, datastore

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * Note: This file is set up differently from the others on purpose.
# *       Every other RMC finds objects by their data ID, so nothing else
# *       on the server can get in the way. SearchObject will find ANY
# *       object which matches, including ones left over from previous
# *       runs and ones from other users on a live server.
# *       unique_data_type does not help here, it restarts at 1 every run
# *       and the old objects are still sitting there. So every test gets
# *       its own random tag and only searches for that. Just ignoring
# *       objects the test did not post works for most tests, but not the
# *       ones which check the total count, since the leftovers still get
# *       counted
# *
# *       Some of the setup is also really slow on live (1010 objects for
# *       the total count tests, timestamp delays for the sorting tests,
# *       etc.) so those objects are posted once for the whole file in
# *       module scoped fixtures, rather than once per test. No coverage
# *       is lost, it just keeps a live run at around 10 minutes rather
# *       than well over 30. anyio_backend has to be module scoped too for
# *       this to work. Since these fixtures post so many objects, they
# *       also clean up after themselves, unlike the other test files

# * The data type of the objects in the everyone fixture. Far past any
# * unique_data_type, so no other test posts objects with it
EVERYONE_DATA_TYPE = constants.INVALID_DATA_TYPE - 2

# * Each object in the sortable fixture has a different place in every
# * column, so the column the results were sorted on shows in their order
SIZES = [30, 10, 50, 20, 70, 40, 60]
NAMES = ["delta", "Bravo", "alpha", "charlie", "Foxtrot", "", "echo"]
DATA_TYPES = [3, 0, 6, 1, 5, 2, 4]
REFERRED_COUNTS = [5, 0, 2, 6, 1, 4, 3]
UPDATE_ORDER = [3, 0, 6, 1, 5, 2, 4]

# * The initial value of rating slot 0 and the ratings FRIEND_B gives it,
# * or None for no rating slots at all
RATINGS = [(0, [5]), (0, [3, 3, 3]), (100, []), (0, [1, 1]), None, (0, [10, 2]), (-50, [-4])]

# * The fewest matches a search counts, see the total count section
COUNTED_MATCHES = 1000
OVER_A_THOUSAND = COUNTED_MATCHES + 10

# * In seconds. How long DataStoreProtocol::SearchObject
# * results are cached for when useCache is set. The real
# * lifetime is not known so this is set intentionall low
# * as to not not slow down the tests.
# * THIS NEEDS TO MATCH THE TEST SERVER VALUE, AND BE UPDATED FOR LIVE MODE
SEARCH_CACHE_LIFETIME = 5

def new_tag() -> str:
	"""
	A tag nothing else has
	"""

	return uuid.uuid4().hex[:constants.MAX_TAG_LENGTH]

def search_param(**fields) -> datastore.DataStoreSearchParam:
	"""
	Creates a DataStoreSearchParam which finds every public object, with no
	other filters. Any field can be overridden with a keyword argument
	"""

	param = datastore.DataStoreSearchParam()
	param.search_target = constants.SearchType.SEARCH_TYPE_PUBLIC
	param.owner_ids = []
	param.owner_type = constants.SearchTarget.SEARCH_TARGET_ANYBODY
	param.destination_ids = []
	param.data_type = constants.INVALID_DATA_TYPE
	param.created_after = common.DateTime(constants.INVALID_DATE_TIME)
	param.created_before = common.DateTime(constants.INVALID_DATE_TIME)
	param.updated_after = common.DateTime(constants.INVALID_DATE_TIME)
	param.updated_before = common.DateTime(constants.INVALID_DATE_TIME)
	param.refer_data_id = constants.INVALID_DATAID
	param.tags = []
	param.result_order_column = constants.SearchSortColumn.SEARCH_SORT_COLUMN_DATAID
	param.result_order = constants.SearchSortOrder.SEARCH_SORT_ORDER_ASC
	param.result_range = common.ResultRange(0, constants.MAX_SEARCH_RESULT_SIZE)
	param.result_option = 0
	param.minimal_rating_frequency = 0
	param.use_cache = False
	param.total_count_enabled = True
	param.data_types = []

	return helpers.override(param, fields)

async def search(pid: int, **fields) -> datastore.DataStoreSearchResult:
	"""
	Searches for objects as the given user. Each field is a field of the param
	"""

	async with helpers.connect(pid) as client:
		return await client.search_object(search_param(**fields))

async def search_error(pid: int, **fields) -> str:
	"""
	Searches for objects and expects an error to be thrown, returning the
	name of the error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup
	async with helpers.connect(pid) as client:
		try:
			await client.search_object(search_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::SearchObject did not throw an error")

async def found(pid: int, **fields) -> list[int]:
	"""
	The data IDs of the objects a search finds, in the order they came back
	"""

	return [info.data_id for info in (await search(pid, **fields)).result]

async def post(pid: int, tag: str, **fields) -> int:
	"""
	Posts an object as the given user, with the given tag as well as any
	other tags given
	"""

	return await helpers.post_meta_binary(pid, tags=[tag] + fields.pop("tags", []), **fields)

async def post_over_time(tag: str, count: int) -> list[int]:
	"""
	Posts objects with the given tag, far enough apart that each one was
	created at a different time
	"""

	data_ids = []

	for i in range(count):
		if i != 0:
			await anyio.sleep(helpers.TIMESTAMP_DELAY)

		data_ids.append(await post(nex.FRIEND_A, tag))

	return data_ids

async def delete_every(pid: int, data_ids: list[int]) -> None:
	"""
	Deletes objects the given user posted, in batches. Used to clean up after
	the module scoped fixtures, which post a lot of objects
	"""

	async with helpers.connect(pid) as client:
		for start in range(0, len(data_ids), constants.BATCH_PROCESSING_CAPACITY):
			batch = data_ids[start:start + constants.BATCH_PROCESSING_CAPACITY]

			await client.delete_objects([helpers.delete_param(data_id=data_id) for data_id in batch], False)

@pytest.fixture
def tag() -> str:
	"""
	A tag nothing else has. Each test gives it to the objects it posts and
	searches for it, so only that tests objects can be found
	"""

	return new_tag()

@pytest.fixture(scope="module")
def anyio_backend() -> str:
	"""
	Module scoped, so the module scoped fixtures below can be async. They
	post a lot of objects, so they are only posted once for the whole file
	"""

	return "asyncio"

# * Every object in the everyone fixture. "A.specified->B" is FRIEND_A's object
# * which is specified to FRIEND_B
EVERYONE = [
	(nex.FRIEND_A, "A.public", constants.Permission.PERMISSION_PUBLIC, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.friend", constants.Permission.PERMISSION_FRIEND, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.specified->B", constants.Permission.PERMISSION_SPECIFIED, [nex.FRIEND_B], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.specified->S", constants.Permission.PERMISSION_SPECIFIED, [nex.STRANGER], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.specified->B+S", constants.Permission.PERMISSION_SPECIFIED, [nex.FRIEND_B, nex.STRANGER], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.specified_friend->B", constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_B], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.specified_friend->S", constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.STRANGER], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.private", constants.Permission.PERMISSION_PRIVATE, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_A, "A.pending", constants.Permission.PERMISSION_PUBLIC, [], constants.DataFlag.DATA_FLAG_NEED_REVIEW),
	(nex.FRIEND_A, "A.pending.private", constants.Permission.PERMISSION_PRIVATE, [], constants.DataFlag.DATA_FLAG_NEED_REVIEW),
	(nex.FRIEND_B, "B.public", constants.Permission.PERMISSION_PUBLIC, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_B, "B.friend", constants.Permission.PERMISSION_FRIEND, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_B, "B.specified->A", constants.Permission.PERMISSION_SPECIFIED, [nex.FRIEND_A], constants.DataFlag.DATA_FLAG_NONE),
	(nex.FRIEND_B, "B.private", constants.Permission.PERMISSION_PRIVATE, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.STRANGER, "S.public", constants.Permission.PERMISSION_PUBLIC, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.STRANGER, "S.friend", constants.Permission.PERMISSION_FRIEND, [], constants.DataFlag.DATA_FLAG_NONE),
	(nex.STRANGER, "S.specified->A", constants.Permission.PERMISSION_SPECIFIED, [nex.FRIEND_A], constants.DataFlag.DATA_FLAG_NONE),
	(nex.STRANGER, "S.specified_friend->A", constants.Permission.PERMISSION_SPECIFIED_FRIEND, [nex.FRIEND_A], constants.DataFlag.DATA_FLAG_NONE),
	(nex.STRANGER, "S.private", constants.Permission.PERMISSION_PRIVATE, [], constants.DataFlag.DATA_FLAG_NONE)
]

@pytest.fixture(scope="module")
async def everyone():
	"""
	An object of every permission from each user, plus FRIEND_A's objects
	which are pending review and rejected, all with EVERYONE_DATA_TYPE.
	Returns the name of each one by data ID
	"""

	names = {}
	posted = []

	for pid, name, permission, recipients, flag in EVERYONE:
		data_id = await helpers.post_meta_binary(pid, data_type=EVERYONE_DATA_TYPE, name=name, permission=helpers.permission(permission=permission, recipients=recipients), flag=flag)
		names[data_id] = name
		posted.append((pid, data_id))

	rejected = await helpers.post_rejected(helpers.post_meta_binary, nex.FRIEND_A, data_type=EVERYONE_DATA_TYPE, name="A.rejected")
	names[rejected] = "A.rejected"
	posted.append((nex.FRIEND_A, rejected))

	yield names

	for pid in [nex.FRIEND_A, nex.FRIEND_B, nex.STRANGER]:
		await delete_every(pid, [data_id for owner, data_id in posted if owner == pid])

async def found_names(pid: int, everyone: dict[int, str], **fields) -> list[str]:
	"""
	The names of the objects from the everyone fixture which a search finds,
	sorted. The search is limited to their data type, and anything else of
	that type is left out
	"""

	# * Scoped by data type rather than a tag. Any search of
	# * SEARCH_TYPE_RECEIVED with tags fails with PythonCore::Exception
	response = await search(pid, data_type=EVERYONE_DATA_TYPE, **fields)

	return sorted(everyone[info.data_id] for info in response.result if info.data_id in everyone)

@pytest.fixture(scope="module")
async def sortable():
	"""
	Seven of FRIEND_A's objects with one tag, set up with the lists at the
	top of the file. Returns the tag and their data IDs, in the order they
	were posted
	"""

	tag = new_tag()
	data_ids = []

	for i, (size, name, data_type) in enumerate(zip(SIZES, NAMES, DATA_TYPES)):
		if i != 0:
			await anyio.sleep(helpers.TIMESTAMP_DELAY)

		slots = [] if RATINGS[i] is None else [helpers.rating_slot(initial_value=RATINGS[i][0])]
		data_ids.append(await helpers.post_object(nex.FRIEND_A, size=size, name=name, data_type=data_type, tags=[tag], rating_init_param=slots))

	for i, data_id in enumerate(data_ids):
		if RATINGS[i] is not None:
			for rating_value in RATINGS[i][1]:
				await helpers.rate_object(nex.FRIEND_B, data_id, rating_value)

		await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_REFERRED_COUNT, referred_count=REFERRED_COUNTS[i])

	# * Last, as nothing else may change the update times afterwards
	for i in UPDATE_ORDER:
		await anyio.sleep(helpers.TIMESTAMP_DELAY)
		await helpers.change_meta(nex.FRIEND_A, data_id=data_ids[i], modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME)

	yield tag, data_ids

	await delete_every(nex.FRIEND_A, data_ids)

@pytest.fixture(scope="module")
async def over_a_thousand():
	"""
	1010 of FRIEND_A's objects with one tag, the first 1000 of which also
	have a second tag. Returns both tags, and the data IDs in the order
	they were posted
	"""

	tag = new_tag()
	first_thousand_tag = new_tag()
	data_ids = []

	async with helpers.connect(nex.FRIEND_A) as client:
		for i in range(OVER_A_THOUSAND):
			tags = [tag, first_thousand_tag] if i < COUNTED_MATCHES else [tag]
			data_ids.append(await client.post_meta_binary(helpers.post_param(tags=tags)))

	yield tag, first_thousand_tag, data_ids

	await delete_every(nex.FRIEND_A, data_ids)

# * What comes back

async def test_finds_an_object(tag):
	"""
	Basic functionality test
	"""

	data_id = await post(nex.FRIEND_A, tag)
	response = await search(nex.FRIEND_A, tags=[tag])

	assert [info.data_id for info in response.result] == [data_id]
	assert response.total_count == 1
	assert response.total_count_type == constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_EXACT

async def test_finding_nothing(tag):
	"""
	A search which matches nothing is not an error
	"""

	response = await search(nex.FRIEND_A, tags=[tag])

	assert response.result == []
	assert response.total_count == 0
	assert response.total_count_type == constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_EXACT

async def test_a_result_is_the_objects_metadata(tag):
	"""
	Each result is the objects metadata, the same as GetMeta gives with the
	same result flags. The rating slots come back in slot order
	"""

	result_option = constants.ResultFlag.RESULT_FLAG_TAGS | constants.ResultFlag.RESULT_FLAG_RATINGS | constants.ResultFlag.RESULT_FLAG_METABINARY
	data_id = await helpers.post_object(nex.FRIEND_A, size=helpers.POSTED_SIZE, name="object", data_type=1, meta_binary=b"meta", tags=[tag, "other"], period=30, permission=helpers.permission(permission=constants.Permission.PERMISSION_FRIEND), rating_init_param=[helpers.rating_slot(3, initial_value=7), helpers.rating_slot(0, initial_value=-2)])

	await helpers.rate_object(nex.FRIEND_B, data_id, 4, slot=3)
	await helpers.touch_object(nex.FRIEND_A, data_id=data_id)

	[info] = (await search(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_SEND_FRIEND, tags=[tag], result_option=result_option)).result

	assert info == await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=result_option)

# * searchTarget
# *
# * These search the everyone fixture. Pending and rejected objects are
# * only ever found by the OWN_PENDING, OWN_REJECTED and OWN_ALL search types

@pytest.mark.parametrize("search_type, pid, names", [
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, nex.FRIEND_A, ["A.public", "B.public", "S.public"], id="public"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_FRIEND, nex.FRIEND_A, ["A.friend"], id="send-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, nex.FRIEND_A, ["A.specified->B", "A.specified->B+S", "A.specified->S"], id="send-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED_FRIEND, nex.FRIEND_A, ["A.specified_friend->B", "A.specified_friend->S"], id="send-specified-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, nex.FRIEND_A, ["A.friend", "A.specified->B", "A.specified->B+S", "A.specified->S", "A.specified_friend->B", "A.specified_friend->S"], id="send"),
	pytest.param(constants.SearchType.SEARCH_TYPE_FRIEND, nex.FRIEND_A, ["B.friend"], id="friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_FRIEND, nex.STRANGER, [], id="friend-with-no-friends"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, nex.FRIEND_A, ["B.specified->A", "S.specified->A"], id="received-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, nex.FRIEND_B, ["A.specified->B", "A.specified->B+S", "A.specified_friend->B"], id="received-specified-from-a-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, nex.STRANGER, ["A.specified->B+S", "A.specified->S"], id="received-specified-from-a-stranger"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, nex.FRIEND_A, ["B.friend", "B.specified->A", "S.specified->A"], id="received"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, nex.FRIEND_B, ["A.friend", "A.specified->B", "A.specified->B+S", "A.specified_friend->B"], id="received-from-a-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PRIVATE, nex.FRIEND_A, ["A.private"], id="private"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN, nex.FRIEND_A, ["A.friend", "A.private", "A.public", "A.specified->B", "A.specified->B+S", "A.specified->S", "A.specified_friend->B", "A.specified_friend->S"], id="own"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC_EXCLUDE_OWN_AND_FRIENDS, nex.FRIEND_A, ["S.public"], id="public-exclude-own-and-friends"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC_EXCLUDE_OWN_AND_FRIENDS, nex.STRANGER, ["A.public", "B.public"], id="public-exclude-own-and-friends-with-no-friends"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_PENDING, nex.FRIEND_A, ["A.pending", "A.pending.private"], id="own-pending"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_REJECTED, nex.FRIEND_A, ["A.rejected"], id="own-rejected"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_ALL, nex.FRIEND_A, ["A.friend", "A.pending", "A.pending.private", "A.private", "A.public", "A.rejected", "A.specified->B", "A.specified->B+S", "A.specified->S", "A.specified_friend->B", "A.specified_friend->S"], id="own-all")
])
async def test_search_types(everyone, search_type, pid, names):
	"""
	Each search type finds the objects it names. A specified friend object
	is only received by recipients who are friends of the owner
	"""

	assert await found_names(pid, everyone, search_target=search_type) == names

@pytest.mark.parametrize("search_type", [
	pytest.param(0, id="zero"),
	pytest.param(15, id="past-the-last")
])
async def test_search_types_which_are_not_real(tag, search_type):
	"""
	Only search types 1 to 14 exist
	"""

	assert await search_error(nex.FRIEND_A, search_target=search_type, tags=[tag]) == "DataStore::InvalidArgument"

# * ownerIds and ownerType

@pytest.mark.parametrize("search_type, owner_ids, names", [
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [nex.FRIEND_B], ["B.public"], id="one-owner"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [nex.FRIEND_A, nex.STRANGER], ["A.public", "S.public"], id="several-owners"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, [nex.FRIEND_B], ["B.friend", "B.specified->A"], id="on-top-of-the-search-type"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, [nex.FRIEND_B], [], id="an-owner-the-search-type-leaves-out"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [], ["A.public", "B.public", "S.public"], id="nobody")
])
async def test_owner_ids(everyone, search_type, owner_ids, names):
	"""
	Only objects owned by one of the users listed are found, out of the ones
	the search type finds. An empty list does not filter anything
	"""

	assert await found_names(nex.FRIEND_A, everyone, search_target=search_type, owner_ids=owner_ids) == names

@pytest.mark.parametrize("search_type, names", [
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, ["B.public"], id="public"),
	pytest.param(constants.SearchType.SEARCH_TYPE_FRIEND, ["B.friend"], id="friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, ["B.specified->A"], id="received-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, ["B.friend", "B.specified->A"], id="received")
])
async def test_owner_type_friend(everyone, search_type, names):
	"""
	Only objects owned by the callers friends are found
	"""

	assert await found_names(nex.FRIEND_A, everyone, search_target=search_type, owner_type=constants.SearchTarget.SEARCH_TARGET_FRIEND) == names

async def test_owner_type_friend_with_owner_ids(tag):
	"""
	ownerIds cannot be given, not even a friend
	"""

	assert await search_error(nex.FRIEND_A, owner_type=constants.SearchTarget.SEARCH_TARGET_FRIEND, owner_ids=[nex.FRIEND_B], tags=[tag]) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("search_type", [
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_FRIEND, id="send-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, id="send-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED_FRIEND, id="send-specified-friend"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, id="send"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PRIVATE, id="private"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN, id="own"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC_EXCLUDE_OWN_AND_FRIENDS, id="public-exclude-own-and-friends"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_PENDING, id="own-pending"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_REJECTED, id="own-rejected"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_ALL, id="own-all")
])
async def test_owner_type_friend_with_other_search_types(tag, search_type):
	"""
	Only the public, friend and received search types can be limited to
	friends
	"""

	assert await search_error(nex.FRIEND_A, search_target=search_type, owner_type=constants.SearchTarget.SEARCH_TARGET_FRIEND, tags=[tag]) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("search_type, owner_ids, names", [
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [nex.FRIEND_B], ["A.public", "S.public"], id="one-owner"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, [nex.FRIEND_B], ["S.specified->A"], id="on-top-of-the-search-type"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [], ["A.public", "B.public", "S.public"], id="nobody")
])
async def test_owner_type_exclude(everyone, search_type, owner_ids, names):
	"""
	Objects owned by the users listed are left out
	"""

	assert await found_names(nex.FRIEND_A, everyone, search_target=search_type, owner_type=constants.SearchTarget.SEARCH_TARGET_ANYBODY_EXCLUDE_SPECIFIED, owner_ids=owner_ids) == names

async def test_owner_types_which_are_not_real(tag):
	"""
	Only owner types 0 to 2 exist
	"""

	assert await search_error(nex.FRIEND_A, owner_type=3, tags=[tag]) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("owner_type", [
	pytest.param(constants.SearchTarget.SEARCH_TARGET_ANYBODY, id="anybody"),
	pytest.param(constants.SearchTarget.SEARCH_TARGET_ANYBODY_EXCLUDE_SPECIFIED, id="exclude")
])
@pytest.mark.parametrize("search_type", [
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN, id="own"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_PENDING, id="own-pending"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_REJECTED, id="own-rejected"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_ALL, id="own-all")
])
async def test_own_search_types_with_owner_ids(tag, search_type, owner_type):
	"""
	The own search types cannot be given ownerIds, not even the caller
	"""

	assert await search_error(nex.FRIEND_A, search_target=search_type, owner_type=owner_type, owner_ids=[nex.FRIEND_A], tags=[tag]) == "DataStore::InvalidArgument"

async def test_own_search_types_excluding_nobody(everyone):
	"""
	Excluding nobody is the same as not filtering by owner
	"""

	without = await found_names(nex.FRIEND_A, everyone, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL)

	assert await found_names(nex.FRIEND_A, everyone, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, owner_type=constants.SearchTarget.SEARCH_TARGET_ANYBODY_EXCLUDE_SPECIFIED) == without

# * destinationIds

@pytest.mark.parametrize("search_type, destination_ids, names", [
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, [nex.FRIEND_B], ["A.specified->B", "A.specified->B+S"], id="one-recipient"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, [nex.FRIEND_B, nex.STRANGER], ["A.specified->B", "A.specified->B+S", "A.specified->S"], id="several-recipients"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, [nex.STRANGER], ["A.specified->B+S", "A.specified->S", "A.specified_friend->S"], id="send"),
	pytest.param(constants.SearchType.SEARCH_TYPE_OWN_ALL, [nex.FRIEND_B], ["A.specified->B", "A.specified->B+S", "A.specified_friend->B"], id="own-all"),
	pytest.param(constants.SearchType.SEARCH_TYPE_PUBLIC, [nex.FRIEND_B], [], id="public"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, [nex.FRIEND_A], [], id="the-caller"),
	pytest.param(constants.SearchType.SEARCH_TYPE_SEND, [], ["A.friend", "A.specified->B", "A.specified->B+S", "A.specified->S", "A.specified_friend->B", "A.specified_friend->S"], id="nobody")
])
async def test_destination_ids(everyone, search_type, destination_ids, names):
	"""
	Only objects specified to at least one of the users listed are found,
	once each. Objects without recipients are left out. An empty list does
	not filter anything
	"""

	assert await found_names(nex.FRIEND_A, everyone, search_target=search_type, destination_ids=destination_ids) == names

async def test_destination_ids_ignore_delete_recipients(tag):
	"""
	Only the recipients of the access permission are matched, not the
	recipients of the delete permission
	"""

	data_id = await post(nex.FRIEND_A, tag, delete_permission=helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B]))

	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, tags=[tag]) == [data_id]
	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, tags=[tag], destination_ids=[nex.FRIEND_B]) == []

@pytest.mark.parametrize("search_type", [
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, id="received-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, id="received")
])
async def test_received_destination_ids_naming_the_caller(everyone, search_type):
	"""
	The received search types already only find objects the caller was
	sent, so naming the caller changes nothing
	"""

	without = await found_names(nex.FRIEND_A, everyone, search_target=search_type)

	assert await found_names(nex.FRIEND_A, everyone, search_target=search_type, destination_ids=[nex.FRIEND_A]) == without

@pytest.mark.parametrize("destination_ids", [
	pytest.param([nex.FRIEND_B], id="someone-else"),
	pytest.param([nex.FRIEND_A, nex.FRIEND_B], id="the-caller-and-someone-else"),
	pytest.param([nex.FRIEND_A, nex.FRIEND_A], id="the-caller-twice")
])
@pytest.mark.parametrize("search_type", [
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, id="received-specified"),
	pytest.param(constants.SearchType.SEARCH_TYPE_RECEIVED, id="received")
])
async def test_received_destination_ids_naming_anything_else(search_type, destination_ids):
	"""
	The received search types only allow destinationIds to be the caller
	alone
	"""

	assert await search_error(nex.FRIEND_A, search_target=search_type, data_type=EVERYONE_DATA_TYPE, destination_ids=destination_ids) == "DataStore::InvalidArgument"

async def test_long_owner_and_destination_lists(everyone):
	"""
	There is no limit on how many users ownerIds and destinationIds can list
	"""

	others = list(range(1000000000, 1000000000 + 199))

	assert await found_names(nex.FRIEND_A, everyone, owner_ids=others + [nex.FRIEND_A]) == ["A.public"]
	assert await found_names(nex.FRIEND_A, everyone, search_target=constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, destination_ids=others + [nex.FRIEND_B]) == ["A.specified->B", "A.specified->B+S"]

# * dataType

@pytest.mark.parametrize("data_type, names", [
	pytest.param(1, ["one"], id="a-data-type"),
	pytest.param(0, ["zero"], id="zero"),
	pytest.param(constants.INVALID_DATA_TYPE, ["zero", "one"], id="invalid-data-type")
])
async def test_data_type(tag, data_type, names):
	"""
	Only objects of the data type are found. 0 is a data type like any
	other, and INVALID_DATA_TYPE finds every data type
	"""

	posted = {
		"zero": await post(nex.FRIEND_A, tag, data_type=0),
		"one": await post(nex.FRIEND_A, tag, data_type=1)
	}

	assert await found(nex.FRIEND_A, tags=[tag], data_type=data_type) == [posted[name] for name in names]

# * Dates
# *
# * Each filter includes objects created or updated at exactly the time
# * given. A date of NEVER or INVALID_DATE_TIME leaves the filter unset

async def test_created_after(tag):
	"""
	Objects created at or after the time are found
	"""

	first, second, third = await post_over_time(tag, 3)
	created = (await helpers.get_meta(nex.FRIEND_A, data_id=second)).create_time

	assert await found(nex.FRIEND_A, tags=[tag], created_after=created) == [second, third]

async def test_created_before(tag):
	"""
	Objects created at or before the time are found
	"""

	first, second, third = await post_over_time(tag, 3)
	created = (await helpers.get_meta(nex.FRIEND_A, data_id=second)).create_time

	assert await found(nex.FRIEND_A, tags=[tag], created_before=created) == [first, second]

async def test_created_after_and_before_the_same_time(tag):
	"""
	Only objects created at exactly that time are found
	"""

	first, second, third = await post_over_time(tag, 3)
	created = (await helpers.get_meta(nex.FRIEND_A, data_id=second)).create_time

	assert await found(nex.FRIEND_A, tags=[tag], created_after=created, created_before=created) == [second]

async def test_updated_after(tag):
	"""
	Uses when the object was last updated, not when it was created
	"""

	first, second = await post_over_time(tag, 2)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await helpers.change_meta(nex.FRIEND_A, data_id=first, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME)

	updated = (await helpers.get_meta(nex.FRIEND_A, data_id=first)).update_time

	assert await found(nex.FRIEND_A, tags=[tag], updated_after=updated) == [first]

async def test_updated_before(tag):
	"""
	Uses when the object was last updated, not when it was created
	"""

	first, second = await post_over_time(tag, 2)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await helpers.change_meta(nex.FRIEND_A, data_id=first, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_UPDATED_TIME)

	updated = (await helpers.get_meta(nex.FRIEND_A, data_id=second)).update_time

	assert await found(nex.FRIEND_A, tags=[tag], updated_before=updated) == [second]

@pytest.mark.parametrize("fields", [
	pytest.param({"created_after": common.DateTime(constants.NEVER)}, id="created-after-never"),
	pytest.param({"created_after": common.DateTime(constants.INVALID_DATE_TIME)}, id="created-after-invalid-date-time"),
	pytest.param({"created_before": common.DateTime(constants.NEVER)}, id="created-before-never"),
	pytest.param({"created_before": common.DateTime(constants.INVALID_DATE_TIME)}, id="created-before-invalid-date-time"),
	pytest.param({"updated_after": common.DateTime(constants.NEVER)}, id="updated-after-never"),
	pytest.param({"updated_after": common.DateTime(constants.INVALID_DATE_TIME)}, id="updated-after-invalid-date-time"),
	pytest.param({"updated_before": common.DateTime(constants.NEVER)}, id="updated-before-never"),
	pytest.param({"updated_before": common.DateTime(constants.INVALID_DATE_TIME)}, id="updated-before-invalid-date-time")
])
async def test_unset_dates(tag, fields):
	"""
	NEVER and INVALID_DATE_TIME leave any of the date filters unset
	"""

	data_id = await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], **fields) == [data_id]

async def test_permanent_date_time_is_a_real_date(tag):
	"""
	PERMANENT_DATE_TIME (9999-12-31 00:00:00) filters like any other date
	"""

	await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], created_after=common.DateTime(constants.PERMANENT_DATE_TIME)) == []

@pytest.mark.parametrize("fields", [
	pytest.param({"created_after": common.DateTime.make(2021), "created_before": common.DateTime.make(2020)}, id="created"),
	pytest.param({"updated_after": common.DateTime.make(2021), "updated_before": common.DateTime.make(2020)}, id="updated")
])
async def test_a_range_which_ends_before_it_starts(tag, fields):
	"""
	Impossible range, so the search is refused
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], **fields) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("fields", [
	pytest.param({"created_after": common.DateTime(constants.INVALID_DATE_TIME), "created_before": common.DateTime.make(2100)}, id="created-unset-start"),
	pytest.param({"created_after": common.DateTime.make(2000), "created_before": common.DateTime(constants.NEVER)}, id="created-unset-end"),
	pytest.param({"updated_after": common.DateTime(constants.INVALID_DATE_TIME), "updated_before": common.DateTime.make(2100)}, id="updated-unset-start"),
	pytest.param({"updated_after": common.DateTime.make(2000), "updated_before": common.DateTime(constants.NEVER)}, id="updated-unset-end")
])
async def test_a_range_with_an_unset_end(tag, fields):
	"""
	Only a range with both ends set is checked for ending before it starts
	"""

	data_id = await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], **fields) == [data_id]

async def test_created_and_updated_ranges_are_not_compared(tag):
	"""
	Created after a time and updated before an earlier one is not an error,
	it just finds nothing
	"""

	await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], created_after=common.DateTime.make(2100), updated_before=common.DateTime.make(2000)) == []

@pytest.mark.parametrize("date", [
	pytest.param(common.DateTime(1), id="just-past-never"),
	pytest.param(common.DateTime.make(0), id="year-0"),
	pytest.param(common.DateTime.make(10000), id="year-10000"),
	pytest.param(common.DateTime.make(2020, 0, 1), id="month-0"),
	pytest.param(common.DateTime.make(2020, 13), id="month-13"),
	pytest.param(common.DateTime.make(2020, 1, 0), id="day-0"),
	pytest.param(common.DateTime.make(2020, 2, 30), id="february-30"),
	pytest.param(common.DateTime.make(2021, 2, 29), id="february-29-not-in-a-leap-year"),
	pytest.param(common.DateTime.make(1900, 2, 29), id="february-29-in-a-century-which-is-not-a-leap-year"),
	pytest.param(common.DateTime.make(2020, 4, 31), id="april-31"),
	pytest.param(common.DateTime.make(2020, 1, 1, 24), id="hour-24"),
	pytest.param(common.DateTime.make(2020, 1, 1, 0, 60), id="minute-60"),
	pytest.param(common.DateTime.make(2020, 1, 1, 0, 0, 60), id="second-60")
])
async def test_dates_which_are_not_real(tag, date):
	"""
	A date which is not a real date cannot be used
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], created_after=date) == "PythonCore::ConversionError"

@pytest.mark.parametrize("year", [
	pytest.param(2020, id="a-leap-year"),
	pytest.param(2000, id="a-century-which-is-a-leap-year")
])
async def test_february_29_in_a_leap_year(tag, year):
	"""
	February 29 is a real date some years
	"""

	data_id = await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], created_after=common.DateTime.make(year, 2, 29)) == [data_id]

@pytest.mark.parametrize("fields", [
	pytest.param({"created_before": common.DateTime.make(2020, 13)}, id="created-before"),
	pytest.param({"updated_after": common.DateTime.make(2020, 13)}, id="updated-after"),
	pytest.param({"updated_before": common.DateTime.make(2020, 13)}, id="updated-before")
])
async def test_every_date_is_checked(tag, fields):
	"""
	Each of the date filters must be a real date, not just createdAfter
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], **fields) == "PythonCore::ConversionError"

# * referDataId

async def test_refer_data_id(tag):
	"""
	Only objects which refer to the object are found
	"""

	referred = await post(nex.FRIEND_A, tag)
	referring = await post(nex.FRIEND_A, tag, refer_data_id=referred)
	other_data_type = await post(nex.FRIEND_A, tag, refer_data_id=referred, data_type=1)

	await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], refer_data_id=referred) == [referring, other_data_type]

async def test_refer_data_id_invalid_dataid(tag):
	"""
	INVALID_DATAID leaves the filter unset, so objects which refer to
	another are found as well as those which do not
	"""

	referred = await post(nex.FRIEND_A, tag)
	referring = await post(nex.FRIEND_A, tag, refer_data_id=referred)

	assert await found(nex.FRIEND_A, tags=[tag], refer_data_id=constants.INVALID_DATAID) == [referred, referring]

async def test_refer_data_id_of_nothing(tag):
	"""
	Nothing refers to an object which does not exist
	"""

	await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag], refer_data_id=helpers.UNKNOWN_DATA_ID) == []

# * tags

@pytest.mark.parametrize("tags, names", [
	pytest.param(["x"], ["x y", "x"], id="one-tag"),
	pytest.param(["x", "y"], ["x y"], id="every-tag-must-match"),
	pytest.param(["y", "x"], ["x y"], id="in-any-order"),
	pytest.param(["x", "z"], [], id="one-which-does-not-match"),
	pytest.param(["x", "x"], ["x y", "x"], id="repeated")
])
async def test_tags(tag, tags, names):
	"""
	An object is found when it has every tag searched for, whatever other
	tags it has. The order of the tags, and repeats, make no difference
	"""

	posted = {
		"x y": await post(nex.FRIEND_A, tag, tags=["x", "y"]),
		"x": await post(nex.FRIEND_A, tag, tags=["x"]),
		"y z": await post(nex.FRIEND_A, tag, tags=["y", "z"])
	}

	assert await found(nex.FRIEND_A, tags=[tag] + tags) == [posted[name] for name in names]

@pytest.mark.parametrize("posted, searched", [
	pytest.param("abc", "ABC", id="case"),
	pytest.param("abc", "ab", id="the-start"),
	pytest.param("abc", "b", id="the-middle"),
	pytest.param("abc", "a%", id="percent-wildcard"),
	pytest.param("abc", "a_c", id="underscore-wildcard"),
	pytest.param("abc", " abc", id="leading-space"),
	pytest.param("e", "é", id="accent")
])
async def test_tags_are_matched_exactly(tag, posted, searched):
	"""
	A tag only matches the same tag, with nothing treated as a wildcard
	"""

	await post(nex.FRIEND_A, tag, tags=[posted])

	assert await found(nex.FRIEND_A, tags=[tag, searched]) == []

@pytest.mark.parametrize("posted, searched", [
	pytest.param("abc ", "abc", id="on-the-object"),
	pytest.param("abc", "abc ", id="in-the-search")
])
async def test_trailing_spaces_are_ignored(tag, posted, searched):
	"""
	Tags which only differ by spaces at the end match each other, whichever
	side has them
	"""

	data_id = await post(nex.FRIEND_A, tag, tags=[posted])

	assert await found(nex.FRIEND_A, tags=[tag, searched]) == [data_id]

async def test_an_empty_tag(tag):
	"""
	An empty tag is matched like any other
	"""

	data_id = await post(nex.FRIEND_A, tag, tags=[""])

	assert await found(nex.FRIEND_A, tags=[tag, ""]) == [data_id]

async def test_a_tag_too_long_to_post(tag):
	"""
	The length of a tag is not checked, it just matches nothing
	"""

	await post(nex.FRIEND_A, tag)

	assert await found(nex.FRIEND_A, tags=[tag, "t" * (constants.MAX_TAG_LENGTH + 1)]) == []

async def test_as_many_tags_as_an_object_can_have(tag):
	"""
	An object can have NUM_TAG_SLOT tags, and all of them can be searched for
	"""

	tags = ["tag%d" % i for i in range(constants.NUM_TAG_SLOT - 1)]
	data_id = await post(nex.FRIEND_A, tag, tags=tags)

	assert await found(nex.FRIEND_A, tags=[tag] + tags) == [data_id]

@pytest.mark.parametrize("tags", [
	pytest.param(["tag%d" % i for i in range(constants.NUM_TAG_SLOT)], id="different-tags"),
	pytest.param(["tag%d" % i for i in range(constants.NUM_TAG_SLOT - 1)] + ["tag0"], id="one-repeated")
])
async def test_too_many_tags(tag, tags):
	"""
	At most NUM_TAG_SLOT tags can be searched for, counting repeats
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag] + tags) == "DataStore::InvalidArgument"

async def test_no_tags(unique_data_type, tag):
	"""
	Searching for no tags does not filter by tag
	"""

	data_id = await post(nex.FRIEND_A, tag, data_type=unique_data_type)

	assert data_id in await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, data_type=unique_data_type)

# * resultOrderColumn and resultOrder

@pytest.mark.parametrize("column, order", [
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_DATAID, [0, 1, 2, 3, 4, 5, 6], id="data-id"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_SIZE, [1, 3, 0, 5, 2, 6, 4], id="size"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_NAME_ALPHABETICAL, [5, 2, 1, 3, 0, 6, 4], id="name"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_DATA_TYPE, [1, 3, 5, 0, 6, 4, 2], id="data-type"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_REFERRED_COUNT, [1, 4, 2, 6, 5, 0, 3], id="referred-count"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_CREATED_TIME, [0, 1, 2, 3, 4, 5, 6], id="created-time"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_UPDATED_TIME, [3, 0, 6, 1, 5, 2, 4], id="updated-time"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0, [6, 3, 0, 1, 5, 2], id="rating"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0, [6, 2, 3, 1, 0, 5], id="rating-average")
])
async def test_sort_columns(sortable, column, order):
	"""
	Each column sorts on what it is named after, in either order. Names are
	compared ignoring case. The rating columns leave out objects without the
	slot, and a slot nobody has rated averages 0
	"""

	tag, data_ids = sortable
	ascending = await found(nex.FRIEND_A, tags=[tag], result_order_column=column)
	descending = await found(nex.FRIEND_A, tags=[tag], result_order_column=column, result_order=constants.SearchSortOrder.SEARCH_SORT_ORDER_DESC)

	assert ascending == [data_ids[i] for i in order]
	assert descending == ascending[::-1]

async def test_sort_by_owner(tag):
	"""
	SEARCH_SORT_COLUMN_OWNER_ID sorts by the PID of the owner
	"""

	posted = {pid: await post(pid, tag) for pid in [nex.FRIEND_A, nex.FRIEND_B, nex.STRANGER]}
	ascending = [posted[pid] for pid in sorted(posted)]

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_OWNER_ID) == ascending
	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_OWNER_ID, result_order=constants.SearchSortOrder.SEARCH_SORT_ORDER_DESC) == ascending[::-1]

@pytest.mark.parametrize("order", [
	pytest.param(constants.SearchSortOrder.SEARCH_SORT_ORDER_ASC, id="ascending"),
	pytest.param(constants.SearchSortOrder.SEARCH_SORT_ORDER_DESC, id="descending")
])
async def test_ties_are_broken_by_data_id(tag, order):
	"""
	Objects which sort the same come back oldest data ID first, whichever
	the order. Names which only differ by case or accent sort the same
	"""

	data_ids = [await post(nex.FRIEND_A, tag, name=name) for name in ["é", "E", "e"]]

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_NAME_ALPHABETICAL, result_order=order) == data_ids

@pytest.mark.parametrize("slot", [
	pytest.param(slot, id="slot-%d" % slot) for slot in range(constants.NUM_RATING_SLOT)
])
async def test_rating_columns_sort_on_their_own_slot(tag, slot):
	"""
	Each rating column sorts on the total of its own slot, and each rating
	average column on the average of it. Every object has every slot, but
	only the slot being sorted on differs between them, so sorting on any
	other slot would tie and come back in data ID order
	"""

	unrated = await helpers.post_rated(nex.FRIEND_A, slots=[helpers.rating_slot(i, initial_value=100 if i == slot else 0) for i in range(constants.NUM_RATING_SLOT)], tags=[tag])
	rated_once = await helpers.post_rated(nex.FRIEND_A, slots=helpers.rating_slots(constants.NUM_RATING_SLOT), tags=[tag])
	rated_four_times = await helpers.post_rated(nex.FRIEND_A, slots=helpers.rating_slots(constants.NUM_RATING_SLOT), tags=[tag])

	await helpers.rate_object(nex.FRIEND_B, rated_once, 5, slot=slot)

	for _ in range(4):
		await helpers.rate_object(nex.FRIEND_B, rated_four_times, 1, slot=slot)

	# * Totals of 100, 5 and 4, and averages of 0, 5 and 1
	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0 + slot) == [rated_four_times, rated_once, unrated]
	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0 + slot) == [unrated, rated_four_times, rated_once]

@pytest.mark.parametrize("column, names", [
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING15, ["slot 15", "both"], id="rating"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE15, ["both", "slot 15"], id="rating-average")
])
async def test_rating_columns_only_find_objects_with_the_slot(tag, column, names):
	"""
	Objects without the slot the column is for are left out. Neither object
	with slot 15 has been rated, so they average the same, and sort by data ID
	"""

	posted = {
		"slot 0": await post(nex.FRIEND_A, tag, rating_init_param=[helpers.rating_slot(0)]),
		"both": await post(nex.FRIEND_A, tag, rating_init_param=[helpers.rating_slot(0), helpers.rating_slot(15, initial_value=7)]),
		"slot 15": await post(nex.FRIEND_A, tag, rating_init_param=[helpers.rating_slot(15, initial_value=3)]),
		"none": await post(nex.FRIEND_A, tag)
	}

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=column) == [posted[name] for name in names]

async def test_a_rating_average_leaves_out_the_initial_value(tag):
	"""
	The average is of the ratings given. A slot which has not been rated
	averages 0, whatever its initial value
	"""

	given_two = await helpers.post_rated(nex.FRIEND_A, slots=[helpers.rating_slot(initial_value=100)], tags=[tag])
	given_three = await helpers.post_rated(nex.FRIEND_A, slots=[helpers.rating_slot()], tags=[tag])
	not_rated = await helpers.post_rated(nex.FRIEND_A, slots=[helpers.rating_slot(initial_value=50)], tags=[tag])

	await helpers.rate_object(nex.FRIEND_B, given_two, 2)
	await helpers.rate_object(nex.FRIEND_B, given_three, 3)

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0) == [not_rated, given_two, given_three]

@pytest.mark.parametrize("first_ratings, second_ratings", [
	pytest.param([3, 4], [3], id="a-half"),
	pytest.param([], [0, -1], id="a-negative-half")
])
async def test_a_rating_average_is_not_rounded(tag, first_ratings, second_ratings):
	"""
	The first object averages half a point more than the second, so it sorts
	after it, rather than tying and sorting first by its data ID
	"""

	first = await helpers.post_rated(nex.FRIEND_A, tags=[tag])
	second = await helpers.post_rated(nex.FRIEND_A, tags=[tag])

	for rating_value in first_ratings:
		await helpers.rate_object(nex.FRIEND_B, first, rating_value)

	for rating_value in second_ratings:
		await helpers.rate_object(nex.FRIEND_B, second, rating_value)

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0) == [second, first]

@pytest.mark.parametrize("column", [
	pytest.param(8, id="past-the-owner"),
	pytest.param(63, id="before-the-ratings"),
	pytest.param(80, id="past-the-ratings"),
	pytest.param(95, id="before-the-rating-averages"),
	pytest.param(112, id="past-the-rating-averages")
])
async def test_sort_columns_which_are_not_real(tag, column):
	"""
	The columns are 0 to 7, then a rating and a rating average column for
	each of the 16 rating slots
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], result_order_column=column) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("order", [
	pytest.param(2, id="past-the-last"),
	pytest.param(255, id="largest")
])
async def test_sort_orders_which_are_not_real(tag, order):
	"""
	The only orders are ascending and descending
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], result_order=order) == "DataStore::InvalidArgument"

# * resultRange

@pytest.mark.parametrize("offset, size, start, end", [
	pytest.param(0, 3, 0, 3, id="the-first-page"),
	pytest.param(3, 3, 3, 6, id="the-second-page"),
	pytest.param(6, 3, 6, 7, id="a-page-past-the-end"),
	pytest.param(7, 3, 7, 7, id="an-offset-past-the-end"),
	pytest.param(constants.RESULTRANGE_ANY_OFFSET - 1, 3, 7, 7, id="just-below-any-offset")
])
async def test_result_range(sortable, offset, size, start, end):
	"""
	The range picks which of the results come back. The total count is
	every match, wherever the range is
	"""

	tag, data_ids = sortable
	response = await search(nex.FRIEND_A, tags=[tag], result_range=common.ResultRange(offset, size))

	assert [info.data_id for info in response.result] == data_ids[start:end]
	assert response.total_count == len(data_ids)

@pytest.mark.parametrize("offset, size", [
	pytest.param(0, 0, id="nothing"),
	pytest.param(0, constants.MAX_SEARCH_RESULT_SIZE + 1, id="past-the-most"),
	pytest.param(constants.RESULTRANGE_ANY_OFFSET, 0, id="nothing-from-any-offset"),
	pytest.param(constants.RESULTRANGE_ANY_OFFSET, constants.MAX_SEARCH_ANY_RESULT_SIZE + 1, id="past-the-most-from-any-offset")
])
async def test_result_range_sizes_which_are_not_allowed(tag, offset, size):
	"""
	At least one result must be asked for, and at most MAX_SEARCH_RESULT_SIZE,
	or MAX_SEARCH_ANY_RESULT_SIZE from RESULTRANGE_ANY_OFFSET
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], result_range=common.ResultRange(offset, size)) == "DataStore::InvalidArgument"

@pytest.mark.parametrize("size", [
	pytest.param(3, id="some"),
	pytest.param(constants.MAX_SEARCH_ANY_RESULT_SIZE, id="more-than-there-are")
])
async def test_any_offset(sortable, size):
	"""
	RESULTRANGE_ANY_OFFSET gives that many of the matches, picked at random
	and in no particular order
	"""

	tag, data_ids = sortable
	response = await search(nex.FRIEND_A, tags=[tag], result_range=common.ResultRange(constants.RESULTRANGE_ANY_OFFSET, size))
	results = [info.data_id for info in response.result]

	assert len(results) == min(size, len(data_ids))
	assert len(set(results)) == len(results)
	assert set(results) <= set(data_ids)
	assert response.total_count == len(data_ids)

# * resultOption

@pytest.mark.parametrize("result_option, filled", [
	pytest.param(0, [], id="none"),
	pytest.param(constants.ResultFlag.RESULT_FLAG_TAGS, ["tags"], id="tags"),
	pytest.param(constants.ResultFlag.RESULT_FLAG_RATINGS, ["ratings"], id="ratings"),
	pytest.param(constants.ResultFlag.RESULT_FLAG_METABINARY, ["meta_binary"], id="meta-binary"),
	pytest.param(0xFF, ["tags", "ratings", "meta_binary"], id="bits-which-are-not-flags")
])
async def test_result_option(tag, result_option, filled):
	"""
	The tags, rating slots and meta binary are only filled in when their
	result flag is set, and are empty otherwise. Bits which are not result
	flags are ignored
	"""

	data_id = await post(nex.FRIEND_A, tag, meta_binary=b"meta", rating_init_param=[helpers.rating_slot(initial_value=5)])
	everything = await helpers.get_meta(nex.FRIEND_A, data_id=data_id, result_option=constants.ResultFlag.RESULT_FLAG_ALL)
	empty = helpers.empty_meta_info()

	[info] = (await search(nex.FRIEND_A, tags=[tag], result_option=result_option)).result

	for field in ["tags", "ratings", "meta_binary"]:
		assert getattr(info, field) == getattr(everything if field in filled else empty, field)

@pytest.mark.parametrize("pid, search_type", [
	pytest.param(nex.FRIEND_A, constants.SearchType.SEARCH_TYPE_SEND_SPECIFIED, id="the-owner"),
	pytest.param(nex.FRIEND_B, constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, id="a-recipient")
])
async def test_recipients_never_come_back(tag, pid, search_type):
	"""
	The permissions only hold the permission itself. The recipients are
	left out, even with RESULT_FLAG_PERMITTED_IDS
	"""

	await post(nex.FRIEND_A, tag, permission=helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B]), delete_permission=helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B]))

	[info] = (await search(pid, search_target=search_type, tags=[tag], result_option=constants.ResultFlag.RESULT_FLAG_ALL)).result

	assert info.permission == helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED)
	assert info.delete_permission == helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED)

# * minimalRatingFrequency

@pytest.mark.parametrize("column, frequency, order", [
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0, 1, [6, 3, 0, 1, 5], id="leaves-out-slots-nobody-rated"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0, 3, [1], id="exactly-enough"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0, 4, [], id="more-than-any-slot-has"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0, 2, [3, 1, 5], id="rating-average")
])
async def test_minimal_rating_frequency(sortable, column, frequency, order):
	"""
	Only objects whose slot has been rated at least that many times are
	found, and counted in the total
	"""

	tag, data_ids = sortable
	response = await search(nex.FRIEND_A, tags=[tag], result_order_column=column, minimal_rating_frequency=frequency)

	assert [info.data_id for info in response.result] == [data_ids[i] for i in order]
	assert response.total_count == len(order)

@pytest.mark.parametrize("slot", [pytest.param(slot, id="slot-%d" % slot) for slot in range(constants.NUM_RATING_SLOT)])
async def test_minimal_rating_frequency_counts_the_columns_slot(tag, slot):
	"""
	Only the ratings of the slot the column is for are counted
	"""

	next_slot = (slot + 1) % constants.NUM_RATING_SLOT
	rated_once = await helpers.post_rated(nex.FRIEND_A, slots=helpers.rating_slots(constants.NUM_RATING_SLOT), tags=[tag])
	rated_twice = await helpers.post_rated(nex.FRIEND_A, slots=helpers.rating_slots(constants.NUM_RATING_SLOT), tags=[tag])
	rated_on_the_next_slot = await helpers.post_rated(nex.FRIEND_A, slots=helpers.rating_slots(constants.NUM_RATING_SLOT), tags=[tag])

	await helpers.rate_object(nex.FRIEND_B, rated_once, 1, slot=slot)

	for _ in range(2):
		await helpers.rate_object(nex.FRIEND_B, rated_twice, 1, slot=slot)
		await helpers.rate_object(nex.FRIEND_B, rated_on_the_next_slot, 1, slot=next_slot)

	for _ in range(3):
		await helpers.rate_object(nex.FRIEND_B, rated_once, 1, slot=next_slot)

	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING0 + slot, minimal_rating_frequency=2) == [rated_twice]
	assert await found(nex.FRIEND_A, tags=[tag], result_order_column=constants.SearchSortColumn.SEARCH_SORT_COLUMN_RATING_AVERAGE0 + slot, minimal_rating_frequency=2) == [rated_twice]

@pytest.mark.parametrize("column", [
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_DATAID, id="data-id"),
	pytest.param(constants.SearchSortColumn.SEARCH_SORT_COLUMN_NAME_ALPHABETICAL, id="name")
])
async def test_minimal_rating_frequency_without_a_rating_column(tag, column):
	"""
	There is no slot to count the ratings of
	"""

	assert await search_error(nex.FRIEND_A, tags=[tag], result_order_column=column, minimal_rating_frequency=1) == "DataStore::InvalidArgument"

# * The newer fields
# *
# * useCache, totalCountEnabled and dataTypes were added in later revisions of
# * the param during NEX 3. NEX 4 just set the revision counter back to 0 and
# * uses the latest revision. Going by the DDLs of every Wii U game, useCache
# * was added somewhere between NEX 3.7.1 and 3.8.2, totalCountEnabled between
# * 3.8.3 and 3.9.1, and dataTypes between 3.9.1 and 3.10.0. The checks below
# * use the earliest version known to have each field, so they may be off for
# * versions inside those ranges. The only live NEX server on Nintendo Network
# * at the time of writing is Pokemon Bank, which is NEX 3.4 and predates all
# * of them, so only the side where they are ignored has been tested on live
# * servers

@pytest.mark.parametrize("use_cache", [
	pytest.param(True, id="cached"),
	pytest.param(False, id="not-cached")
])
async def test_use_cache(tag, use_cache):
	"""
	With useCache, a search identical to an earlier one gives the same
	results as before until the cache expires, even if the objects have
	changed since. Without it, a search always sees the objects as they
	are now
	"""

	# * How the cache is formed is not known. This assumes the cache is keyed on
	# * every field of the param. The exact time the cache exists for is also
	# * not known, so SEARCH_CACHE_LIFETIME is set to be very small just for
	# * testing
	data_id = await post(nex.FRIEND_A, tag, name="before")

	await search(nex.FRIEND_A, tags=[tag], use_cache=use_cache)
	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_NAME, name="after")

	[info] = (await search(nex.FRIEND_A, tags=[tag], use_cache=use_cache)).result

	if use_cache and nex.SERVER_NEX_VERSION >= 30802:
		assert info.name == "before"

		# * A second longer, so the cache has definitely expired
		await anyio.sleep(SEARCH_CACHE_LIFETIME + 1)

		[info] = (await search(nex.FRIEND_A, tags=[tag], use_cache=use_cache)).result

	assert info.name == "after"

async def test_total_count_enabled(tag):
	"""
	Turning totalCountEnabled off leaves out the total count
	"""

	await post(nex.FRIEND_A, tag)
	response = await search(nex.FRIEND_A, tags=[tag], total_count_enabled=False)

	if nex.SERVER_NEX_VERSION < 30901:
		assert response.total_count == 1
		assert response.total_count_type == constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_EXACT
	else:
		assert response.total_count == 0
		assert response.total_count_type == constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_DISABLED

async def test_data_types(tag):
	"""
	Objects of the data types listed are found as well as those of dataType
	"""

	first = await post(nex.FRIEND_A, tag, data_type=1)
	second = await post(nex.FRIEND_A, tag, data_type=2)

	await post(nex.FRIEND_A, tag, data_type=3)

	results = await found(nex.FRIEND_A, tags=[tag], data_type=1, data_types=[2])

	if nex.SERVER_NEX_VERSION < 31000:
		assert results == [first]
	else:
		assert results == [first, second]

# * Total count
# *
# * Only the first max(1000, offset + 5 * size) matches are counted. When
# * more than that match, the total count is how many were counted, and is
# * only a minimum

async def test_a_thousand_matches_are_counted_exactly(over_a_thousand):
	"""
	1000 matches are all counted, so the total is exact
	"""

	_, first_thousand_tag, _ = over_a_thousand
	response = await search(nex.FRIEND_A, tags=[first_thousand_tag], result_range=common.ResultRange(0, 1))

	assert response.total_count == COUNTED_MATCHES
	assert response.total_count_type == constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_EXACT

@pytest.mark.parametrize("result_range, total_count, total_count_type", [
	pytest.param(common.ResultRange(0, constants.MAX_SEARCH_RESULT_SIZE), 1000, constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_MINIMUM, id="a-thousand-at-the-least"),
	pytest.param(common.ResultRange(999, 1), 1004, constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_MINIMUM, id="five-pages-past-the-offset"),
	pytest.param(common.ResultRange(1000, 2), 1010, constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_EXACT, id="reaching-every-match"),
	pytest.param(common.ResultRange(constants.RESULTRANGE_ANY_OFFSET, 1), 1000, constants.SearchResultTotalCountType.SEARCH_RESULT_TOTAL_MINIMUM, id="any-offset")
])
async def test_over_a_thousand_matches(over_a_thousand, result_range, total_count, total_count_type):
	"""
	The page asked for decides how many of the 1010 matches are counted
	"""

	tag, _, _ = over_a_thousand
	response = await search(nex.FRIEND_A, tags=[tag], result_range=result_range)

	assert response.total_count == total_count
	assert response.total_count_type == total_count_type

async def test_paging_past_the_counted_matches(over_a_thousand):
	"""
	Matches past the ones counted can still be paged to
	"""

	tag, _, data_ids = over_a_thousand

	assert await found(nex.FRIEND_A, tags=[tag], result_range=common.ResultRange(COUNTED_MATCHES, 10)) == data_ids[COUNTED_MATCHES:]

# * Objects which are not finished, or have changed

@pytest.mark.parametrize("prepare_post", [
	pytest.param(helpers.prepare_post_object, id="prepare-post-object"),
	pytest.param(helpers.prepare_post_object_v1, id="prepare-post-object-v1")
])
async def test_an_upload_which_was_never_completed(tag, prepare_post):
	"""
	An object is not found until its upload is completed, not even by its
	owner
	"""

	await prepare_post(nex.FRIEND_A, tags=[tag])

	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, tags=[tag]) == []

async def test_a_failed_upload(tag):
	"""
	An upload which was completed as a failure is never found
	"""

	async with helpers.connect(nex.FRIEND_A) as client:
		post_info = await client.prepare_post_object(helpers.post_param(tags=[tag]))
		await client.complete_post_object(helpers.complete_post_param(data_id=post_info.data_id, success=False))

	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, tags=[tag]) == []

async def test_a_deleted_object(tag):
	"""
	Deleted objects are never found
	"""

	data_id = await post(nex.FRIEND_A, tag)

	await helpers.delete_object(nex.FRIEND_A, data_id=data_id)

	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_ALL, tags=[tag]) == []

async def test_an_object_being_updated(tag):
	"""
	Until the update is completed, the object is found as it was
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, size=helpers.POSTED_SIZE, tags=[tag])
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await helpers.prepare_update_object(nex.FRIEND_A, data_id=data_id, size=helpers.UPDATED_SIZE)

	[info] = (await search(nex.FRIEND_A, tags=[tag])).result

	assert info.size == helpers.POSTED_SIZE
	assert info.update_time == before.update_time

async def test_an_updated_object(tag):
	"""
	Once the update is completed, the object is found with its new size and
	update time
	"""

	data_id = await helpers.post_object(nex.FRIEND_A, size=helpers.POSTED_SIZE, tags=[tag])
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)

	update_info = await helpers.prepare_update_object(nex.FRIEND_A, data_id=data_id, size=helpers.UPDATED_SIZE)
	await helpers.complete_update_object(nex.FRIEND_A, data_id=data_id, version=update_info.version)

	[info] = (await search(nex.FRIEND_A, tags=[tag])).result

	assert info.size == helpers.UPDATED_SIZE
	assert info.update_time.timestamp() > before.update_time.timestamp()

async def test_changed_tags(tag):
	"""
	Objects are found by the tags they have now
	"""

	data_id = await post(nex.FRIEND_A, tag, tags=["before"])

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_TAGS, tags=[tag, "after"])

	assert await found(nex.FRIEND_A, tags=[tag, "before"]) == []
	assert await found(nex.FRIEND_A, tags=[tag, "after"]) == [data_id]

async def test_changed_recipients(tag):
	"""
	Objects are received by the recipients they have now
	"""

	data_id = await post(nex.FRIEND_A, tag, permission=helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.FRIEND_B]))

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_ACCESS_PERMISSION, permission=helpers.permission(permission=constants.Permission.PERMISSION_SPECIFIED, recipients=[nex.STRANGER]))

	assert await found(nex.FRIEND_B, search_target=constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, tags=[tag]) == []
	assert await found(nex.STRANGER, search_target=constants.SearchType.SEARCH_TYPE_RECEIVED_SPECIFIED, tags=[tag]) == [data_id]

async def test_an_approved_object(tag):
	"""
	Once an object pending review is approved, others can find it, and it
	is no longer pending
	"""

	data_id = await post(nex.FRIEND_A, tag, flag=constants.DataFlag.DATA_FLAG_NEED_REVIEW)

	assert await found(nex.FRIEND_B, tags=[tag]) == []

	await helpers.change_meta(nex.FRIEND_A, data_id=data_id, modifies_flag=constants.ModificationFlag.MODIFICATION_FLAG_STATUS, status=constants.DataStatus.DATA_STATUS_NONE)

	assert await found(nex.FRIEND_B, tags=[tag]) == [data_id]
	assert await found(nex.FRIEND_A, search_target=constants.SearchType.SEARCH_TYPE_OWN_PENDING, tags=[tag]) == []

async def test_searching_is_not_a_reference(tag):
	"""
	Finding an object leaves its reference count and referred time alone
	"""

	data_id = await post(nex.FRIEND_A, tag)
	before = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	await anyio.sleep(helpers.TIMESTAMP_DELAY)
	await search(nex.FRIEND_B, tags=[tag])

	after = await helpers.get_meta(nex.FRIEND_A, data_id=data_id)

	assert after.referred_count == before.referred_count
	assert after.referred_time == before.referred_time
