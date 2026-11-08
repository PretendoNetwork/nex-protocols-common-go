import itertools

import pytest

_data_types = itertools.count(1)

@pytest.fixture
def unique_data_type():
	"""
	Returns a data type no other test uses. Objects posted with it, and
	searches filtered by it, will not see objects from other tests
	"""

	return next(_data_types)
