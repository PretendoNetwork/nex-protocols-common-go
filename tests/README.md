# Tests

Integration tests for the common protocols. Most of this library needs a running server, a database, and a real client to test, so rather than using `_test.go` files the tests run against a real NEX server and uses [NintendoClients](https://github.com/kinnay/NintendoClients) as the testing client.

## Running

```bash
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/pytest
```

## Live mode

By default the tests run against a local test server, which is built and started automatically. Live mode runs them against a real, live, NEX server instead. This is useful to run the test suite against a real Nintendo server to ensure accuracy.

> [!CAUTION]
> Live mode operates on a real server, using real accounts. Tests may modify data in unexpected ways, create garbage data, use RMCs in unexpected ways, etc. All tests are written to try and not cause any lasting damage/side effects to the live server, and to prevent bans, however use at your own risk. Using an account you actively use may result in real data loss, and tests which trip red flags for Nintendo may result in an account/console ban.

> [!NOTE]
> Running in live mode is often orders of magnitude slower than local tests, due to network latency.

### Setup

Live mode needs a server and three accounts, which are given as environment variables:

| Variable                                    | Description                                                                                                                                                                        |
|---------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `PN_NEX_COMMON_TEST_LIVE_HOST`              | Address of the server's authentication server                                                                                                                                      |
| `PN_NEX_COMMON_TEST_LIVE_PORT`              | Port of the server's authentication server                                                                                                                                         |
| `PN_NEX_COMMON_TEST_LIVE_ACCESS_KEY`        | The game's NEX access key                                                                                                                                                          |
| `PN_NEX_COMMON_TEST_LIVE_NEX_VERSION`       | The game's NEX version, in the form NintendoClients uses, such as `30402` for NEX 3.4.2                                                                                            |
| `PN_NEX_COMMON_TEST_LIVE_FRIEND_A_USERNAME` | PID of the first account. This account is also the "main" account used by tests                                                                                                    |
| `PN_NEX_COMMON_TEST_LIVE_FRIEND_A_PASSWORD` | Password of the first account                                                                                                                                                      |
| `PN_NEX_COMMON_TEST_LIVE_FRIEND_B_USERNAME` | PID of the second account, which must be friends with the first. This account is also used as the "secondary" account                                                              |
| `PN_NEX_COMMON_TEST_LIVE_FRIEND_B_PASSWORD` | Password of the second account                                                                                                                                                     |
| `PN_NEX_COMMON_TEST_LIVE_STRANGER_USERNAME` | PID of the third account, which must not be friends with either of the others. This account is used to sanity check things like permission settings which are gated by friendships |
| `PN_NEX_COMMON_TEST_LIVE_STRANGER_PASSWORD` | Password of the third account                                                                                                                                                      |

These can also go in a `.env` file, either in this folder or in the project root. This folder is checked first, and anything already set in the environment takes priority over the file. For example, for Pokemon Bank:

```bash
PN_NEX_COMMON_TEST_LIVE_HOST=52.40.192.64
PN_NEX_COMMON_TEST_LIVE_PORT=56000
PN_NEX_COMMON_TEST_LIVE_ACCESS_KEY=9a2961d8
PN_NEX_COMMON_TEST_LIVE_NEX_VERSION=30407
PN_NEX_COMMON_TEST_LIVE_FRIEND_A_USERNAME=...
PN_NEX_COMMON_TEST_LIVE_FRIEND_A_PASSWORD=...
PN_NEX_COMMON_TEST_LIVE_FRIEND_B_USERNAME=...
PN_NEX_COMMON_TEST_LIVE_FRIEND_B_PASSWORD=...
PN_NEX_COMMON_TEST_LIVE_STRANGER_USERNAME=...
PN_NEX_COMMON_TEST_LIVE_STRANGER_PASSWORD=...
```

### Running

Set `PN_NEX_COMMON_TEST_LIVE=1` for the run after setting the above variables:

```bash
PN_NEX_COMMON_TEST_LIVE=1 .venv/bin/pytest
```

> [!CAUTION]
> Don't put `PN_NEX_COMMON_TEST_LIVE` in the `.env` file, or every run will go live.

> [!NOTE]
> Some tests may not run in live mode, since not every NEX server supports every feature. Certain features may be skipped in cases such as incompatible NEX versions.
