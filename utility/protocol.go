package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

var commonUtilityProtocol *CommonUtilityProtocol

type CommonUtilityProtocol struct {
	*utility.Protocol
	server              *nex.PRUDPServer
	GenerateNEXUniqueID func() uint64
}

// NewCommonUtilityProtocol returns a new CommonUtilityProtocol
func NewCommonUtilityProtocol(server *nex.PRUDPServer) *CommonUtilityProtocol {
	utilityProtocol := utility.NewProtocol(server)
	commonUtilityProtocol = &CommonUtilityProtocol{Protocol: utilityProtocol, server: server}

	// TODO - Organize these by method ID
	commonUtilityProtocol.AcquireNexUniqueID = acquireNexUniqueID

	return commonUtilityProtocol
}
