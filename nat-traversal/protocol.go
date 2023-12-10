package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

var commonNATTraversalProtocol *CommonNATTraversalProtocol

type CommonNATTraversalProtocol struct {
	server   nex.ServerInterface
	protocol nat_traversal.Interface
}

// NewCommonNATTraversalProtocol returns a new CommonNATTraversalProtocol
func NewCommonNATTraversalProtocol(protocol nat_traversal.Interface) *CommonNATTraversalProtocol {
	protocol.SetHandlerRequestProbeInitiationExt(requestProbeInitiationExt)
	protocol.SetHandlerReportNATProperties(reportNATProperties)
	protocol.SetHandlerReportNATTraversalResult(reportNATTraversalResult)
	protocol.SetHandlerGetRelaySignatureKey(getRelaySignatureKey)
	protocol.SetHandlerReportNATTraversalResultDetail(reportNATTraversalResultDetail)

	commonNATTraversalProtocol = &CommonNATTraversalProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonNATTraversalProtocol
}
