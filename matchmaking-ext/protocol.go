package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
)

var commonMatchMakingExtProtocol *CommonMatchMakingExtProtocol

type CommonMatchMakingExtProtocol struct {
	*match_making_ext.Protocol
	server *nex.PRUDPServer
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchMakingExtProtocol(server *nex.PRUDPServer) *CommonMatchMakingExtProtocol {
	MatchMakingExtProtocol := match_making_ext.NewProtocol(server)
	commonMatchMakingExtProtocol = &CommonMatchMakingExtProtocol{Protocol: MatchMakingExtProtocol, server: server}

	MatchMakingExtProtocol.EndParticipation(endParticipation)

	return commonMatchMakingExtProtocol
}
