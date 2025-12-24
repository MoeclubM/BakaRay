# BakaRay
A panel to manage forward rules.

## Setup

1. Copy `.env.example` to `.env` and tweak database/Redis/site values. `DB_TYPE` controls whether SQLite (default) or MySQL/MariaDB is used; the SQL credentials (`DB_HOST`, `DB_PORT`, `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`) are only required for the latter.
2. Build the server with `go build ./cmd/server`.
3. Create an initial user via the provided script:
   ```bash
   go run ./cmd/init-account --username admin --password some-secret
   ```
   The script reads the same configuration as the server and will refuse to run unless both username and password are supplied.
