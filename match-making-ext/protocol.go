package match_making_ext

import (
	"github.com/PretendoNetwork/nex-go"
	match_making_ext "github.com/PretendoNetwork/nex-protocols-go/match-making-ext"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol match_making_ext.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making_ext.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerEndParticipation(commonProtocol.endParticipation)

	return commonProtocol
}
