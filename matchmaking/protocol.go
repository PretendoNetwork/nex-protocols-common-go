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

	GetConnectionUrlsHandler func(rvcid uint32) []string
	UpdateRoomHostHandler    func(gid uint32, newownerpid uint32)
	DestroyRoomHandler       func(gid uint32)
	GetRoomInfoHandler       func(gid uint32) (uint32, uint32, uint32, uint32, uint32)
	GetRoomPlayersHandler    func(gid uint32) []uint32
}

// GetConnectionUrls sets the GetConnectionUrls handler function
func (commonMatchMakingProtocol *CommonMatchMakingProtocol) GetConnectionUrls(handler func(rvcid uint32) []string) {
	commonMatchMakingProtocol.GetConnectionUrlsHandler = handler
}

// UpdateRoomHost sets the UpdateRoomHost handler function
func (commonMatchMakingProtocol *CommonMatchMakingProtocol) UpdateRoomHost(handler func(gid uint32, newownerpid uint32)) {
	commonMatchMakingProtocol.UpdateRoomHostHandler = handler
}

// DestroyRoom sets the DestroyRoom handler function
func (commonMatchMakingProtocol *CommonMatchMakingProtocol) DestroyRoom(handler func(gid uint32)) {
	commonMatchMakingProtocol.DestroyRoomHandler = handler
}

// GetRoomInfo sets the GetRoomInfo handler function
func (commonMatchMakingProtocol *CommonMatchMakingProtocol) GetRoomInfo(handler func(gid uint32) (uint32, uint32, uint32, uint32, uint32)) {
	commonMatchMakingProtocol.GetRoomInfoHandler = handler
}

// GetRoomPlayers sets the GetRoomPlayers handler function
func (commonMatchMakingProtocol *CommonMatchMakingProtocol) GetRoomPlayers(handler func(gid uint32) []uint32) {
	commonMatchMakingProtocol.GetRoomPlayersHandler = handler
}

// NewCommonMatchMakingProtocol returns a new CommonMatchMakingProtocol
func NewCommonMatchMakingProtocol(server *nex.Server) *CommonMatchMakingProtocol {
	matchMakingProtocol := match_making.NewMatchMakingProtocol(server)
	commonMatchMakingProtocol = &CommonMatchMakingProtocol{MatchMakingProtocol: matchMakingProtocol, server: server}

	commonMatchMakingProtocol.GetSessionURLs(getSessionURLs)
	commonMatchMakingProtocol.UnregisterGathering(unregisterGathering)
	commonMatchMakingProtocol.UpdateSessionHostV1(updateSessionHostV1)
	commonMatchMakingProtocol.UpdateSessionHost(updateSessionHost)

	if server.PRUDPVersion() == 0 {
		server.On("Kick", func(packet *nex.PacketV0) {
			fmt.Println("Leaving")
			common_globals.RemoveConnectionIDFromAllSessions(packet.Sender().ConnectionID())
		})
	} else {
		server.On("Kick", func(packet *nex.PacketV1) {
			fmt.Println("Leaving")
			common_globals.RemoveConnectionIDFromAllSessions(packet.Sender().ConnectionID())
		})
	}

	return commonMatchMakingProtocol
}
