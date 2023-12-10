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
	*match_making.Protocol
	server *nex.PRUDPServer
}

// NewCommonMatchMakingProtocol returns a new CommonMatchMakingProtocol
func NewCommonMatchMakingProtocol(server *nex.PRUDPServer) *CommonMatchMakingProtocol {
	matchMakingProtocol := match_making.NewProtocol(server)
	commonMatchMakingProtocol = &CommonMatchMakingProtocol{Protocol: matchMakingProtocol, server: server}

	common_globals.Sessions = make(map[uint32]*common_globals.CommonMatchmakeSession)

	// TODO - Organize these by method ID
	commonMatchMakingProtocol.UpdateSessionURL = updateSessionURL
	commonMatchMakingProtocol.GetSessionURLs = getSessionURLs
	commonMatchMakingProtocol.UnregisterGathering = unregisterGathering
	commonMatchMakingProtocol.UpdateSessionHostV1 = updateSessionHostV1
	commonMatchMakingProtocol.UpdateSessionHost = updateSessionHost
	commonMatchMakingProtocol.FindBySingleID = findBySingleID

	// TODO - Once websockets are implemented, make an interface for PRUDP
	// and websockets which implements this function
	server.OnClientRemoved(func(client *nex.PRUDPClient) {
		fmt.Println("Leaving")
		common_globals.RemoveClientFromAllSessions(client)
	})

	return commonMatchMakingProtocol
}
