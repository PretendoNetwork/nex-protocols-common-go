package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
)

var commonMatchMakingExtProtocol *CommonMatchMakingExtProtocol

type CommonMatchMakingExtProtocol struct {
	server   nex.ServerInterface
	protocol match_making_ext.Interface
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchMakingExtProtocol(protocol match_making_ext.Interface) *CommonMatchMakingExtProtocol {
	protocol.SetHandlerEndParticipation(endParticipation)

	commonMatchMakingExtProtocol = &CommonMatchMakingExtProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonMatchMakingExtProtocol
}
