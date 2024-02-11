package utility

import (
	"github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	endpoint            nex.EndpointInterface
	protocol            utility.Interface
	GenerateNEXUniqueID func() uint64
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol utility.Interface) *CommonProtocol {
	protocol.SetHandlerAcquireNexUniqueID(acquireNexUniqueID)

	commonProtocol = &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	return commonProtocol
}
