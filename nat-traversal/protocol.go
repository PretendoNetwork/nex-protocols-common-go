package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server   nex.ServerInterface
	protocol nat_traversal.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol nat_traversal.Interface) *CommonProtocol {
	protocol.SetHandlerRequestProbeInitiationExt(requestProbeInitiationExt)
	protocol.SetHandlerReportNATProperties(reportNATProperties)
	protocol.SetHandlerReportNATTraversalResult(reportNATTraversalResult)
	protocol.SetHandlerGetRelaySignatureKey(getRelaySignatureKey)
	protocol.SetHandlerReportNATTraversalResultDetail(reportNATTraversalResultDetail)

	commonProtocol = &CommonProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonProtocol
}
