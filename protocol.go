package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

var (
	server                                *nex.Server
	GetPlayerUrlsHandler                  func(pid uint32) []string
	UpdatePlayerSessionUrlHandler         func(pid uint32, oldurl string, newurl string)
	GetPlayerSessionAddressHandler        func(pid uint32) string
)

// GetPlayerUrls sets the GetPlayerUrls handler function
func GetPlayerUrls(handler func(pid uint32) []string) {
	GetPlayerUrlsHandler = handler
}

// UpdatePlayerSessionUrl sets the UpdatePlayerSessionUrl handler function
func UpdatePlayerSessionUrl(handler func(pid uint32, oldurl string, newurl string)) {
	UpdatePlayerSessionUrlHandler = handler
}

// GetPlayerSessionAddress sets the GetPlayerSessionAddress handler function
func GetPlayerSessionAddress(handler func(pid uint32) string) {
	GetPlayerSessionAddressHandler = handler
}

// InitNatTraversalProtocol returns a new NatTraversalProtocol
func InitNatTraversalProtocol(server *nex.Server) *nexproto.NatTraversalProtocol {
	natTraversalProtocolServer := nexproto.NewNatTraversalProtocol(server)

	natTraversalProtocolServer.RequestProbeInitiationExt(requestProbeInitiationExt)
	natTraversalProtocolServer.ReportNATProperties(reportNatProperties)
	natTraversalProtocolServer.ReportNATTraversalResult(reportNATTraversalResult)
	return natTraversalProtocolServer
}