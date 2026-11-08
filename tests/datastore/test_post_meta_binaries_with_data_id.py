import pytest
from nintendo.nex import common

import nex
from datastore import helpers

pytestmark = pytest.mark.anyio

# * DISCLAIMER, THESE TESTS ASSUME THE WIKI IS CORRECT AND HAVE NOT BEEN
# * CHECKED AGAINST A REAL SERVER. THE ONLY LIVE NEX SERVER ON NINTENDO
# * NETWORK AT THE TIME OF WRITING IS POKEMON BANK, WHICH HAS CUSTOM RMCs
# * FROM 40+, SO THIS COULD NOT BE TESTED. THE WIKI SAYS THIS METHOD IS ONLY
# * AVAILABLE TO SERVER OPERATORS, SO THESE TESTS ONLY CHECK THAT A NORMAL
# * CLIENT IS REFUSED. WHICH ERROR IS GIVEN, AND WHETHER IT IS GIVEN FOR THE
# * WHOLE REQUEST OR FOR EACH ENTRY, IS NOT KNOWN

async def post(pid: int, data_ids: list[int], params: list, transactional: bool) -> list[str | None]:
	"""
	Posts objects with set data IDs as the given user. Returns the name of
	each result, where an entry which worked is None. If the whole request
	is refused, every entry gets the name of that error
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			results = await client.post_meta_binaries_with_data_id(data_ids, params, transactional)
		except common.RMCError as error:
			return [error.name()] * len(data_ids)

	return helpers.result_names(results)

# * data_ids

@pytest.mark.parametrize("data_ids", [
	pytest.param([999998, 999999], id="the-system-range"),
	pytest.param([helpers.UNKNOWN_DATA_ID - 1, helpers.UNKNOWN_DATA_ID], id="the-normal-range")
])
@pytest.mark.parametrize("transactional", [True, False])
async def test_a_normal_client_cannot_post(data_ids, transactional):
	"""
	Nothing is posted, whichever range the data IDs are in
	"""

	results = await post(nex.FRIEND_A, data_ids, [helpers.post_param()] * len(data_ids), transactional)

	assert None not in results

	for data_id in data_ids:
		assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

@pytest.mark.parametrize("transactional", [True, False])
async def test_a_data_id_which_is_already_used(unique_data_type, transactional):
	"""
	The object already using the data ID is left alone
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="before")
	results = await post(nex.FRIEND_A, [data_id], [helpers.post_param(data_type=unique_data_type, name="after")], transactional)

	assert None not in results
	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "before"
