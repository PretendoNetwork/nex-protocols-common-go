package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"github.com/PretendoNetwork/plogger-go"
)

var commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol

type CommonMatchmakeExtensionProtocol struct {
	*nexproto.MatchmakeExtensionProtocol
	server *nex.Server

	DestroyRoomHandler                    func(gid uint32)
	GetRoomHandler                        func(gid uint32) (uint32, *nexproto.MatchmakeSession)
	NewRoomHandler                        func(gid uint32, matchmakeSession *nexproto.MatchmakeSession) (uint32)
	FindRoomViaMatchmakeSessionHandler    func(matchmakeSession *nexproto.MatchmakeSession) (uint32)
	AddPlayerToRoomHandler                func(gid uint32, pid uint32, addplayercount uint32)
}

var logger = plogger.NewLogger()

// DestroyRoom sets the DestroyRoom handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) DestroyRoom(handler func(gid uint32)) {
	commonMatchmakeExtensionProtocol.DestroyRoomHandler = handler
}

// GetRoom sets the GetRoom handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) GetRoom(handler func(gid uint32) (uint32, *nexproto.MatchmakeSession)) {
	commonMatchmakeExtensionProtocol.GetRoomHandler = handler
}

// NewRoom sets the NewRoom handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) NewRoom(handler func(gid uint32, matchmakeSession *nexproto.MatchmakeSession) (uint32)) {
	commonMatchmakeExtensionProtocol.NewRoomHandler = handler
}

// GetRoomInfo sets the GetRoomInfo handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) FindRoomViaMatchmakeSession(handler func(matchmakeSession *nexproto.MatchmakeSession) (uint32)) {
	commonMatchmakeExtensionProtocol.FindRoomViaMatchmakeSessionHandler = handler
}

// AddPlayerToRoom sets the AddPlayerToRoomHandler handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) AddPlayerToRoom(handler func(gid uint32, pid uint32, addplayercount uint32)) {
	commonMatchmakeExtensionProtocol.AddPlayerToRoomHandler = handler
}

// NewCommonSecureConnectionProtocol returns a new CommonSecureConnectionProtocol
func NewCommonMatchmakeExtensionProtocol(server *nex.Server) *CommonMatchmakeExtensionProtocol {
	matchmakeExtensionProtocol := nexproto.NewMatchmakeExtensionProtocol(server)
	commonMatchmakeExtensionProtocol = &CommonMatchmakeExtensionProtocol{MatchmakeExtensionProtocol: matchmakeExtensionProtocol, server: server}
	
	commonMatchmakeExtensionProtocol.AutoMatchmake_Postpone(autoMatchmake_Postpone)
	commonMatchmakeExtensionProtocol.AutoMatchmakeWithParam_Postpone(autoMatchmakeWithParam_Postpone)
	commonMatchmakeExtensionProtocol.CreateMatchmakeSessionWithParam(createMatchmakeSessionWithParam)
	commonMatchmakeExtensionProtocol.CreateMatchmakeSession(createMatchmakeSession)
	commonMatchmakeExtensionProtocol.JoinMatchmakeSessionWithParam(joinMatchmakeSessionWithParam)

	return commonMatchmakeExtensionProtocol
}