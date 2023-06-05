package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
	"github.com/PretendoNetwork/plogger-go"
)

var commonNATTraversalProtocol *CommonNATTraversalProtocol
var logger = plogger.NewLogger()

type CommonNATTraversalProtocol struct {
	*nat_traversal.NATTraversalProtocol
	server                      *nex.Server
}

// NewCommonNATTraversalProtocol returns a new CommonNATTraversalProtocol
func NewCommonNATTraversalProtocol(server *nex.Server) *CommonNATTraversalProtocol {
	natTraversalProtocol := nat_traversal.NewNATTraversalProtocol(server)
	commonNATTraversalProtocol = &CommonNATTraversalProtocol{NATTraversalProtocol: natTraversalProtocol, server: server}

	commonNATTraversalProtocol.RequestProbeInitiationExt(requestProbeInitiationExt)
	commonNATTraversalProtocol.ReportNATProperties(reportNATProperties)
	commonNATTraversalProtocol.ReportNATTraversalResult(reportNATTraversalResult)
	return commonNATTraversalProtocol
}
