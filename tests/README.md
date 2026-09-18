# Tests

Integration tests for the common protocols. Most of this library needs a running server, a database, and a real client to test, so rather than using `_test.go` files the tests run against a real NEX server and uses [NintendoClients](https://github.com/kinnay/NintendoClients) as the testing client.

## Running

```bash
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
.venv/bin/pytest
```
