import contextlib

from nintendo.nex import authentication, kerberos, rmc, settings

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
