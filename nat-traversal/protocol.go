package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

var commonNATTraversalProtocol *CommonNATTraversalProtocol

type CommonNATTraversalProtocol struct {
	*nat_traversal.Protocol
	server *nex.Server
}

// NewCommonNATTraversalProtocol returns a new CommonNATTraversalProtocol
func NewCommonNATTraversalProtocol(server *nex.Server) *CommonNATTraversalProtocol {
	natTraversalProtocol := nat_traversal.NewNATTraversalProtocol(server)
	commonNATTraversalProtocol = &CommonNATTraversalProtocol{Protocol: natTraversalProtocol, server: server}

	commonNATTraversalProtocol.RequestProbeInitiationExt(requestProbeInitiationExt)
	commonNATTraversalProtocol.ReportNATProperties(reportNATProperties)
	commonNATTraversalProtocol.ReportNATTraversalResult(reportNATTraversalResult)
	commonNATTraversalProtocol.GetRelaySignatureKey(getRelaySignatureKey)
	commonNATTraversalProtocol.ReportNATTraversalResultDetail(reportNATTraversalResultDetail)
	return commonNATTraversalProtocol
}
