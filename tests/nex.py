import contextlib
import os
import pathlib

from dotenv import load_dotenv
from nintendo.nex import authentication, backend, kerberos, rmc, settings

for folder in [pathlib.Path(__file__).parent, pathlib.Path(__file__).parent.parent]:
	if (folder / ".env").is_file():
		load_dotenv(folder / ".env")
		break

# * Set PN_NEX_COMMON_TEST_LIVE=1 to run the tests against a real live server
# * CAUTION. USE AT YOUR OWN RISK
LIVE = os.environ.get("PN_NEX_COMMON_TEST_LIVE") == "1"

HOST = "127.0.0.1"
DATABASE_PORT = 54329
AUTH_PORT = 60000
SECURE_PORT = 60001

# * These must match tests/server
ACCESS_KEY = "12345678"
NEX_VERSION = 40600
PID_SIZE = 8
PASSWORD = "password"

FRIEND_A = 1000 # * Also used as the main test user
FRIEND_B = 1001
STRANGER = 1002

LIVE_VARIABLES = [
	"PN_NEX_COMMON_TEST_LIVE_HOST",
	"PN_NEX_COMMON_TEST_LIVE_PORT",
	"PN_NEX_COMMON_TEST_LIVE_ACCESS_KEY",
	"PN_NEX_COMMON_TEST_LIVE_NEX_VERSION",
	"PN_NEX_COMMON_TEST_LIVE_FRIEND_A_USERNAME",
	"PN_NEX_COMMON_TEST_LIVE_FRIEND_A_PASSWORD",
	"PN_NEX_COMMON_TEST_LIVE_FRIEND_B_USERNAME",
	"PN_NEX_COMMON_TEST_LIVE_FRIEND_B_PASSWORD",
	"PN_NEX_COMMON_TEST_LIVE_STRANGER_USERNAME",
	"PN_NEX_COMMON_TEST_LIVE_STRANGER_PASSWORD"
]

LIVE_ACCOUNTS = {}

if LIVE:
	missing = [name for name in LIVE_VARIABLES if not os.environ.get(name)]

	if missing:
		raise RuntimeError("PN_NEX_COMMON_TEST_LIVE is set, but these are not: %s" % ", ".join(missing))

	LIVE_HOST = os.environ["PN_NEX_COMMON_TEST_LIVE_HOST"]
	LIVE_PORT = int(os.environ["PN_NEX_COMMON_TEST_LIVE_PORT"])
	LIVE_ACCESS_KEY = os.environ["PN_NEX_COMMON_TEST_LIVE_ACCESS_KEY"]
	LIVE_NEX_VERSION = int(os.environ["PN_NEX_COMMON_TEST_LIVE_NEX_VERSION"])

	for user in ["FRIEND_A", "FRIEND_B", "STRANGER"]:
		username = os.environ["PN_NEX_COMMON_TEST_LIVE_%s_USERNAME" % user]
		LIVE_ACCOUNTS[int(username)] = (username, os.environ["PN_NEX_COMMON_TEST_LIVE_%s_PASSWORD" % user])

	FRIEND_A, FRIEND_B, STRANGER = list(LIVE_ACCOUNTS)

def make_settings() -> settings.Settings:
	s = settings.default()
	s.configure(ACCESS_KEY, NEX_VERSION, 0)
	s["prudp.version"] = 1
	s["nex.pid_size"] = PID_SIZE
	s["nex.struct_header"] = 1

	return s

@contextlib.asynccontextmanager
async def connect(pid: int):
	"""
	Connects a user by PID to the test server
	"""

	if LIVE:
		username, password = LIVE_ACCOUNTS[pid]
		live = settings.default()
		live.configure(LIVE_ACCESS_KEY, LIVE_NEX_VERSION)

		async with backend.connect(live, LIVE_HOST, LIVE_PORT) as be:
			async with be.login(username, password) as client:
				yield client

		return

	s = make_settings()

	# * NintendoClients automatically uses ValidateAndRequestTicketWithParam on
	# * NEX versions higher than 4.4.0. Since we don't support that yet, connect
	# * manually. Ideally our TicketGrantingProtocol implementation should be
	# * updated to support ValidateAndRequestTicketWithParam and this be removed
	async with rmc.connect(s, HOST, AUTH_PORT) as client:
		response = await authentication.AuthenticationClient(client).login(str(pid))
		response.result.raise_if_error()

	key = kerberos.KeyDerivationOld(65000, 1024).derive_key(PASSWORD.encode(), pid)
	ticket = kerberos.ClientTicket.decrypt(response.ticket, key, s)
	station = response.connection_data.main_station
	credentials = kerberos.Credentials(ticket, pid, station["CID"])

	async with rmc.connect(s, station["address"], station["port"], station["sid"], credentials=credentials) as client:
		yield client
