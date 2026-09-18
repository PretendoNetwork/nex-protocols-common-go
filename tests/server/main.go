// Package main implements a NEX server using the common protocols, to be used by the tests
package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	_ "github.com/lib/pq"
)

var database *sql.DB

func main() {
	databasePort := getPort("PN_NEX_COMMON_TEST_DATABASE_PORT", 54329)
	authPort := getPort("PN_NEX_COMMON_TEST_AUTH_PORT", 60000)
	securePort := getPort("PN_NEX_COMMON_TEST_SECURE_PORT", 60001)

	postgres, err := startDatabase(databasePort)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		os.Exit(1)
	}

	authenticationServer := newAuthenticationServer(securePort)
	secureServer := newSecureServer()

	go authenticationServer.Listen(authPort)
	go secureServer.Listen(securePort)

	// * The tests wait for this line before starting
	fmt.Println("READY")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	// * Postgres runs as a separate process, so it has to be stopped before the server exits
	if err := postgres.Stop(); err != nil {
		common_globals.Logger.Error(err.Error())
		os.Exit(1)
	}
}

// getPort reads a port number from the environment, using fallback if it is not set or invalid
func getPort(name string, fallback int) int {
	port, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}

	return port
}
