import logging
import os
import pathlib
import subprocess
import threading
from collections.abc import Iterator

import pytest

import nex

TESTS_PATH = pathlib.Path(__file__).parent
SERVER_SOURCE_PATH = TESTS_PATH / "server"
SERVER_BINARY_PATH = TESTS_PATH / ".bin" / "server"
SERVER_LOG_PATH = TESTS_PATH / "server.log"
SERVER_START_TIMEOUT = 300 # * The first run downloads the Postgres binaries, which can take a while. Bail after 5 minutes and assume something died

def ignore_disconnect_ack_warnings(record: logging.LogRecord) -> bool:
	"""
	Hides the warnings NintendoClients logs when a connection is closed.
	The server sends the DISCONNECT ACK 3 times, like the official servers
	did, but NintendoClients unbinds the port after the first one and warns
	about the other 2
	"""

	return "Port is not bound" not in record.getMessage()

logging.getLogger("nintendo.nex.prudp").addFilter(ignore_disconnect_ack_warnings)

@pytest.fixture(scope="session", autouse=True)
def server_process() -> Iterator[subprocess.Popen[str]]:
	"""
	Builds and starts the test server for the test run
	"""

	subprocess.run(["go", "build", "-o", SERVER_BINARY_PATH, "."], cwd=SERVER_SOURCE_PATH, check=True)

	env = os.environ | {
		"PN_NEX_COMMON_TEST_DATABASE_PORT": str(nex.DATABASE_PORT),
		"PN_NEX_COMMON_TEST_AUTH_PORT": str(nex.AUTH_PORT),
		"PN_NEX_COMMON_TEST_SECURE_PORT": str(nex.SECURE_PORT),
	}

	with open(SERVER_LOG_PATH, "w") as log:
		process = subprocess.Popen([SERVER_BINARY_PATH], cwd=SERVER_BINARY_PATH.parent, env=env, stdout=subprocess.PIPE, stderr=log, text=True)
		ready = threading.Event()

		# * Wait for the "READY" signal from the Go server.
		# * Keep after the signal so the pipe never fills up
		def read_output() -> None:
			for line in process.stdout:
				log.write(line)
				log.flush()

				if line.strip() == "READY":
					ready.set()

		threading.Thread(target=read_output, daemon=True).start()

		if not ready.wait(SERVER_START_TIMEOUT) or process.poll() is not None:
			process.kill()
			pytest.exit(f"Test server failed to start, see {SERVER_LOG_PATH}", returncode=1)

		yield process

		# * SIGTERM lets the server shut down Postgres
		process.terminate()
		process.wait(30)

@pytest.fixture(autouse=True)
def stop_on_server_crash(server_process: subprocess.Popen[str]) -> Iterator[None]:
	"""
	Stops the test run if the server crashed during a test. Otherwise every
	following test fails to connect, hiding the real problem
	"""

	yield

	if server_process.poll() is not None:
		pytest.exit(f"Test server crashed, see {SERVER_LOG_PATH}", returncode=1)

@pytest.fixture
def anyio_backend() -> str:
	return "asyncio"
