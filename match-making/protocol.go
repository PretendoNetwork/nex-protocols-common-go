package matchmaking

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

var commonMatchMakingProtocol *CommonMatchMakingProtocol

type CommonMatchMakingProtocol struct {
	server   *nex.PRUDPServer
	protocol match_making.Interface
}

// NewCommonMatchMakingProtocol returns a new CommonMatchMakingProtocol
func NewCommonMatchMakingProtocol(protocol match_making.Interface) *CommonMatchMakingProtocol {
	// TODO - Remove cast to PRUDPServer once websockets are implemented
	server := protocol.Server().(*nex.PRUDPServer)

	common_globals.Sessions = make(map[uint32]*common_globals.CommonMatchmakeSession)

	protocol.SetHandlerUnregisterGathering(unregisterGathering)
	protocol.SetHandlerFindBySingleID(findBySingleID)
	protocol.SetHandlerUpdateSessionURL(updateSessionURL)
	protocol.SetHandlerUpdateSessionHostV1(updateSessionHostV1)
	protocol.SetHandlerGetSessionURLs(getSessionURLs)
	protocol.SetHandlerUpdateSessionHost(updateSessionHost)

	commonMatchMakingProtocol = &CommonMatchMakingProtocol{
		server:   server,
		protocol: protocol,
	}

	// TODO - Once websockets are implemented, make an interface for PRUDP
	// and websockets which implements this function
	server.OnClientRemoved(func(client *nex.PRUDPClient) {
		fmt.Println("Leaving")
		common_globals.RemoveClientFromAllSessions(client)
	})

	return commonMatchMakingProtocol
}
