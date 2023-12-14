package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server              nex.ServerInterface
	protocol            utility.Interface
	GenerateNEXUniqueID func() uint64
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol utility.Interface) *CommonProtocol {
	protocol.SetHandlerAcquireNexUniqueID(acquireNexUniqueID)

	commonProtocol = &CommonProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonProtocol
}
