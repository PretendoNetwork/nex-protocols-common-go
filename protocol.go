package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

var (
	server                                *nex.Server
	GetConnectionUrlsHandler              func(rvcid uint32) []string
	ReplaceConnectionUrlHandler           func(rvcid uint32, oldurl string, newurl string)
	GetConnectionGlobalAddressHandler     func(rvcid uint32) string
)

// GetConnectionUrls sets the GetConnectionUrls handler function
func GetConnectionUrls(handler func(rvcid uint32) []string) {
	GetConnectionUrlsHandler = handler
}

// ReplaceConnectionUrl sets the ReplaceConnectionUrl handler function
func ReplaceConnectionUrl(handler func(rvcid uint32, oldurl string, newurl string)) {
	ReplaceConnectionUrlHandler = handler
}

// GetGlobalConnectionAddress sets the GetGlobalConnectionAddress handler function
func GetConnectionGlobalAddress(handler func(rvcid uint32) string) {
	GetConnectionGlobalAddressHandler = handler
}

// InitNatTraversalProtocol returns a new NatTraversalProtocol
func InitNatTraversalProtocol(nexServer *nex.Server) *nexproto.NatTraversalProtocol {
	server = nexServer
	natTraversalProtocolServer := nexproto.NewNatTraversalProtocol(nexServer)

	natTraversalProtocolServer.RequestProbeInitiationExt(requestProbeInitiationExt)
	natTraversalProtocolServer.ReportNATProperties(reportNatProperties)
	natTraversalProtocolServer.ReportNATTraversalResult(reportNATTraversalResult)
	return natTraversalProtocolServer
}