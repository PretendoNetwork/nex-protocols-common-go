package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

var commonUtilityProtocol *CommonUtilityProtocol

type CommonUtilityProtocol struct {
	server              nex.ServerInterface
	protocol            utility.Interface
	GenerateNEXUniqueID func() uint64
}

// NewCommonUtilityProtocol returns a new CommonUtilityProtocol
func NewCommonUtilityProtocol(protocol utility.Interface) *CommonUtilityProtocol {
	protocol.SetHandlerAcquireNexUniqueID(acquireNexUniqueID)

	commonUtilityProtocol = &CommonUtilityProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonUtilityProtocol
}
