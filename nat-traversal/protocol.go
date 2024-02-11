package nattraversal

import (
	"github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol nat_traversal.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol nat_traversal.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerRequestProbeInitiationExt(commonProtocol.requestProbeInitiationExt)
	protocol.SetHandlerReportNATProperties(commonProtocol.reportNATProperties)
	protocol.SetHandlerReportNATTraversalResult(commonProtocol.reportNATTraversalResult)
	protocol.SetHandlerGetRelaySignatureKey(commonProtocol.getRelaySignatureKey)
	protocol.SetHandlerReportNATTraversalResultDetail(commonProtocol.reportNATTraversalResultDetail)

	return commonProtocol
}
