package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	"github.com/PretendoNetwork/plogger-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	"fmt"
)

var commonMatchMakingProtocol *CommonMatchMakingProtocol
var logger = plogger.NewLogger()

type CommonMatchMakingProtocol struct {
	*match_making.MatchMakingProtocol
	server *nex.Server
}

// NewCommonMatchMakingProtocol returns a new CommonMatchMakingProtocol
func NewCommonMatchMakingProtocol(server *nex.Server) *CommonMatchMakingProtocol {
	matchMakingProtocol := match_making.NewMatchMakingProtocol(server)
	commonMatchMakingProtocol = &CommonMatchMakingProtocol{MatchMakingProtocol: matchMakingProtocol, server: server}

	common_globals.Sessions = make(map[uint32]*common_globals.CommonMatchmakeSession)

	commonMatchMakingProtocol.GetSessionURLs(getSessionURLs)
	commonMatchMakingProtocol.UnregisterGathering(unregisterGathering)
	commonMatchMakingProtocol.UpdateSessionHostV1(updateSessionHostV1)
	commonMatchMakingProtocol.UpdateSessionHost(updateSessionHost)

	server.On("Kick", func(packet nex.PacketInterface) {
		fmt.Println("Leaving")
		common_globals.RemoveConnectionIDFromAllSessions(packet.Sender().ConnectionID())
	})

	return commonMatchMakingProtocol
}
