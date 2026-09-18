package main

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_datastore "github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	common_ticket_granting "github.com/PretendoNetwork/nex-protocols-common-go/v2/ticket-granting"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

// * Server settings, must match the client settings.
// * In general, these settings are the highest values
// * we can see, to test the most data as possible. Tests
// * for older implementations not handled
const (
	accessKey    = "12345678"
	fragmentSize = 1300
	pidSize      = 8
)

// newServer creates a PRUDP server using the given NEX version
func newServer(version *nex.LibraryVersion) *nex.PRUDPServer {
	server := nex.NewPRUDPServer()

	server.AccessKey = accessKey
	server.SetFragmentSize(fragmentSize)
	server.LibraryVersions.SetDefault(version)
	server.ByteStreamSettings.PIDSize = pidSize
	server.ByteStreamSettings.UseStructureHeader = true

	return server
}

// newAuthenticationServer creates the authentication server.
// testing just this server is currently not possible with NintendoClients
func newAuthenticationServer(securePort int) *nex.PRUDPServer {
	// * NEX 4.0 replaced Login with ValidateAndRequestTicket, which
	// * the common TicketGranting protocol does not implement, so
	// * downgrade to 3.10.2 until that gets added
	server := newServer(nex.NewLibraryVersion(3, 10, 2))
	endpoint := nex.NewPRUDPEndPoint(1)

	endpoint.ServerAccount = authenticationServerAccount
	endpoint.AccountDetailsByPID = accountDetailsByPID
	endpoint.AccountDetailsByUsername = accountDetailsByUsername

	endpoint.OnError(func(err *nex.Error) {
		common_globals.Logger.Errorf("Auth: %v", err)
	})

	ticketGrantingProtocol := ticket_granting.NewProtocol()
	endpoint.RegisterServiceProtocol(ticketGrantingProtocol)
	commonTicketGrantingProtocol := common_ticket_granting.NewCommonProtocol(ticketGrantingProtocol)

	// * Bypass needing a NEX token for tests
	commonTicketGrantingProtocol.EnableInsecureLogin()
	commonTicketGrantingProtocol.SecureServerAccount = secureServerAccount
	commonTicketGrantingProtocol.SecureStationURL = types.NewStationURL(types.NewString(fmt.Sprintf("prudps:/address=127.0.0.1;port=%d;CID=1;PID=2;sid=1;stream=10;type=2", securePort)))
	commonTicketGrantingProtocol.BuildName = types.NewString("nex-protocols-common-go tests")

	server.BindPRUDPEndPoint(endpoint)

	return server
}

// newSecureServer creates the secure server, which implements every common protocol being tested
func newSecureServer() *nex.PRUDPServer {
	server := newServer(nex.NewLibraryVersion(4, 6, 0))
	endpoint := nex.NewPRUDPEndPoint(1)

	endpoint.IsSecureEndPoint = true
	endpoint.ServerAccount = secureServerAccount
	endpoint.AccountDetailsByPID = accountDetailsByPID
	endpoint.AccountDetailsByUsername = accountDetailsByUsername

	endpoint.OnError(func(err *nex.Error) {
		common_globals.Logger.Errorf("Secure: %v", err)
	})

	setupDataStoreCommonProtocol(endpoint)

	server.BindPRUDPEndPoint(endpoint)

	return server
}

func setupDataStoreCommonProtocol(endpoint *nex.PRUDPEndPoint) {
	dataStoreProtocol := datastore.NewProtocol()
	endpoint.RegisterServiceProtocol(dataStoreProtocol)
	commonDataStoreProtocol := common_datastore.NewCommonProtocol(dataStoreProtocol)

	datastoreManager := common_globals.NewDataStoreManager(endpoint, database)
	datastoreManager.GetUserFriendPIDs = getUserFriendPIDs
	datastoreManager.SetS3Config(s3Bucket, s3KeyBase, stubS3Manager{})

	commonDataStoreProtocol.SetManager(datastoreManager)
}
