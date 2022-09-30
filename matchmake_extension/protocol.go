package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"github.com/PretendoNetwork/plogger-go"
)

var (
	server                                *nex.Server
	DestroyRoomHandler                    func(gid uint32)
	GetRoomHandler                        func(gid uint32) (uint32, *nexproto.MatchmakeSession)
	NewRoomHandler                        func(gid uint32, matchmakeSession *nexproto.MatchmakeSession) (uint32)
	FindRoomViaMatchmakeSessionHandler    func(matchmakeSession *nexproto.MatchmakeSession) (uint32)
	AddPlayerToRoomHandler                func(gid uint32, pid uint32, addplayercount uint32)
)

var logger = plogger.NewLogger()

// DestroyRoom sets the DestroyRoom handler function
func DestroyRoom(handler func(gid uint32)) {
	DestroyRoomHandler = handler
}

// GetRoom sets the GetRoom handler function
func GetRoom(handler func(gid uint32) (uint32, *nexproto.MatchmakeSession)) {
	GetRoomHandler = handler
}

// NewRoom sets the NewRoom handler function
func NewRoom(handler func(gid uint32, matchmakeSession *nexproto.MatchmakeSession) (uint32)) {
	NewRoomHandler = handler
}

// GetRoomInfo sets the GetRoomInfo handler function
func FindRoomViaMatchmakeSession(handler func(matchmakeSession *nexproto.MatchmakeSession) (uint32)) {
	FindRoomViaMatchmakeSessionHandler = handler
}

// AddPlayerToRoom sets the AddPlayerToRoomHandler handler function
func AddPlayerToRoom(handler func(gid uint32, pid uint32, addplayercount uint32)) {
	AddPlayerToRoomHandler = handler
}

// InitMatchmakeExtensionProtocol returns a new MatchmakeExtensionProtocol
func InitMatchmakeExtensionProtocol(nexServer *nex.Server) *nexproto.MatchmakeExtensionProtocol {
	server = nexServer
	matchMakingProtocolServer := nexproto.NewMatchmakeExtensionProtocol(nexServer)
	matchMakingProtocolServer.AutoMatchmake_Postpone(autoMatchmake_Postpone)
	matchMakingProtocolServer.AutoMatchmakeWithParam_Postpone(autoMatchmakeWithParam_Postpone)
	matchMakingProtocolServer.CreateMatchmakeSessionWithParam(createMatchmakeSessionWithParam)
	matchMakingProtocolServer.CreateMatchmakeSession(createMatchmakeSession)
	matchMakingProtocolServer.JoinMatchmakeSessionWithParam(joinMatchmakeSessionWithParam)

	return matchMakingProtocolServer
}