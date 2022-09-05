package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"github.com/PretendoNetwork/plogger-go"
)

var (
	server                                *nex.Server
	GetConnectionUrlsHandler              func(rvcid uint32) []string
	UpdateRoomHostHandler                 func(gid uint32, newownerpid uint32)
	DestroyRoomHandler                    func(gid uint32)
	GetRoomInfoHandler                    func(gid uint32) (uint32, uint32, uint32, uint32, uint32)
	GetRoomPlayersHandler                 func(gid uint32) ([]uint32)
)

var logger = plogger.NewLogger()

// GetConnectionUrls sets the GetConnectionUrls handler function
func GetConnectionUrls(handler func(rvcid uint32) []string) {
	GetConnectionUrlsHandler = handler
}

// UpdateRoomHost sets the UpdateRoomHost handler function
func UpdateRoomHost(handler func(gid uint32, newownerpid uint32)) {
	UpdateRoomHostHandler = handler
}

// DestroyRoom sets the DestroyRoom handler function
func DestroyRoom(handler func(gid uint32)) {
	DestroyRoomHandler = handler
}

// GetRoomInfo sets the GetRoomInfo handler function
func GetRoomInfo(handler func(gid uint32) (uint32, uint32, uint32, uint32, uint32)) {
	GetRoomInfoHandler = handler
}

// GetRoomPlayers sets the GetRoomPlayers handler function
func GetRoomPlayers(handler func(gid uint32) ([]uint32)) {
	GetRoomPlayersHandler = handler
}

// InitMatchmakingProtocol returns a new InitMatchmakingProtocol
func InitMatchmakingProtocol(nexServer *nex.Server) *nexproto.MatchMakingProtocol {
	server = nexServer
	matchMakingProtocolServer := nexproto.NewMatchMakingProtocol(nexServer)
	
	matchMakingProtocolServer.GetSessionURLs(getSessionURLs)
	matchMakingProtocolServer.UnregisterGathering(unregisterGathering)
	matchMakingProtocolServer.UpdateSessionHostV1(updateSessionHostV1)
	matchMakingProtocolServer.UpdateSessionHost(updateSessionHost)
	return matchMakingProtocolServer
}