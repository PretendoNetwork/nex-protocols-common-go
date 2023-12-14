package matchmaking

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server   *nex.PRUDPServer
	protocol match_making.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making.Interface) *CommonProtocol {
	// TODO - Remove cast to PRUDPServer?
	server := protocol.Server().(*nex.PRUDPServer)

	common_globals.Sessions = make(map[uint32]*common_globals.CommonMatchmakeSession)

	protocol.SetHandlerUnregisterGathering(unregisterGathering)
	protocol.SetHandlerFindBySingleID(findBySingleID)
	protocol.SetHandlerUpdateSessionURL(updateSessionURL)
	protocol.SetHandlerUpdateSessionHostV1(updateSessionHostV1)
	protocol.SetHandlerGetSessionURLs(getSessionURLs)
	protocol.SetHandlerUpdateSessionHost(updateSessionHost)

	commonProtocol = &CommonProtocol{
		server:   server,
		protocol: protocol,
	}

	server.OnClientRemoved(func(client *nex.PRUDPClient) {
		fmt.Println("Leaving")
		common_globals.RemoveClientFromAllSessions(client)
	})

	return commonProtocol
}
