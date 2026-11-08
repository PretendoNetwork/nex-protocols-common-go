import pytest
from nintendo.nex import common

import nex
from datastore import constants, helpers

pytestmark = pytest.mark.anyio

# * DISCLAIMER, THESE TESTS ASSUME THE WIKI IS CORRECT AND HAVE NOT BEEN
# * CHECKED AGAINST A REAL SERVER. THE ONLY LIVE NEX SERVER ON NINTENDO
# * NETWORK AT THE TIME OF WRITING IS POKEMON BANK, WHICH HAS CUSTOM RMCs
# * FROM 40+, SO THIS COULD NOT BE TESTED. THE WIKI SAYS THIS METHOD IS ONLY
# * AVAILABLE TO SERVER OPERATORS, SO THESE TESTS ONLY CHECK THAT A NORMAL
# * CLIENT IS REFUSED. WHICH ERROR IS GIVEN IS NOT KNOWN

async def post_error(pid: int, data_id: int, **fields) -> str:
	"""
	Posts an object with a set data ID as the given user and expects an
	error to be thrown, returning the name of the error. Each field is a
	field of the param
	"""

	# * The error has to be caught inside the connection. If it leaves the
	# * connection NintendoClients wraps it in an ExceptionGroup, and so
	# * does anything else raised in here, such as a failed assertion
	async with helpers.connect(pid) as client:
		try:
			await client.post_meta_binary_with_data_id(data_id, helpers.post_param(**fields))
		except common.RMCError as error:
			return error.name()

	pytest.fail("DataStoreProtocol::PostMetaBinaryWithDataId did not throw an error")

# * data_id

@pytest.mark.parametrize("data_id", [
	pytest.param(999999, id="the-system-range"),
	pytest.param(helpers.UNKNOWN_DATA_ID, id="the-normal-range")
])
async def test_a_normal_client_cannot_post(data_id):
	"""
	Nothing is posted, whichever range the data ID is in
	"""

	await post_error(nex.FRIEND_A, data_id)

	assert await helpers.get_meta_error(nex.FRIEND_A, data_id=data_id) == "DataStore::NotFound"

async def test_a_data_id_which_is_already_used(unique_data_type):
	"""
	The object already using the data ID is left alone
	"""

	data_id = await helpers.post_meta_binary(nex.FRIEND_A, data_type=unique_data_type, name="before")

	await post_error(nex.FRIEND_A, data_id, data_type=unique_data_type, name="after")

	assert (await helpers.get_meta(nex.FRIEND_A, data_id=data_id)).name == "before"

async def test_a_data_id_of_zero():
	"""
	INVALID_DATAID names no object, so there is nothing to post as
	"""

	await post_error(nex.FRIEND_A, constants.INVALID_DATAID)
