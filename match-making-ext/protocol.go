package match_making_ext

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server   nex.ServerInterface
	protocol match_making_ext.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making_ext.Interface) *CommonProtocol {
	protocol.SetHandlerEndParticipation(endParticipation)

	commonProtocol = &CommonProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonProtocol
}
