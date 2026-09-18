package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

// startDatabase starts a new, empty, temporary Postgres server and connects to it
func startDatabase(port int) (*embeddedpostgres.EmbeddedPostgres, error) {
	runtimePath := filepath.Join(os.TempDir(), fmt.Sprintf("nex-protocols-common-go-tests-%d", port))

	// * Sanity check, check if a Postgres server is still running from a previous run and yeet it
	if _, err := os.Stat(filepath.Join(runtimePath, "data", "postmaster.pid")); err == nil {
		exec.Command(filepath.Join(runtimePath, "bin", "pg_ctl"), "stop", "-D", filepath.Join(runtimePath, "data"), "-m", "immediate").Run()
	}

	if err := os.RemoveAll(runtimePath); err != nil {
		return nil, err
	}

	postgresConfig := embeddedpostgres.DefaultConfig().Port(uint32(port)).RuntimePath(runtimePath).Logger(os.Stderr)
	postgres := embeddedpostgres.NewDatabase(postgresConfig)

	if err := postgres.Start(); err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=postgres password=postgres dbname=postgres sslmode=disable", port))
	if err != nil {
		postgres.Stop()
		return nil, err
	}

	if err := db.Ping(); err != nil {
		postgres.Stop()
		return nil, err
	}

	database = db

	return postgres, nil
}
