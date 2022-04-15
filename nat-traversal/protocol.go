package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"github.com/PretendoNetwork/plogger"
)

var (
	server                      *nex.Server
	GetConnectionUrlsHandler    func(rvcid uint32) []string
	ReplaceConnectionUrlHandler func(rvcid uint32, oldurl string, newurl string)
)

var logger = plogger.NewLogger()

// GetConnectionUrls sets the GetConnectionUrls handler function
func GetConnectionUrls(handler func(rvcid uint32) []string) {
	GetConnectionUrlsHandler = handler
}

// ReplaceConnectionUrl sets the ReplaceConnectionUrl handler function
func ReplaceConnectionUrl(handler func(rvcid uint32, oldurl string, newurl string)) {
	ReplaceConnectionUrlHandler = handler
}

// InitNatTraversalProtocol returns a new NatTraversalProtocol
func InitNatTraversalProtocol(nexServer *nex.Server) *nexproto.NATTraversalProtocol {
	server = nexServer
	natTraversalProtocolServer := nexproto.NewNATTraversalProtocol(nexServer)

	natTraversalProtocolServer.RequestProbeInitiationExt(requestProbeInitiationExt)
	natTraversalProtocolServer.ReportNATProperties(reportNatProperties)
	//natTraversalProtocolServer.ReportNATTraversalResult(reportNATTraversalResult) // not implemented in nex-protocols-go yet
	return natTraversalProtocolServer
}
