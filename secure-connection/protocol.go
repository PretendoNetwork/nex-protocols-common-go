package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

var (
	server                                *nex.Server
	secureProtocolServer                  *nexproto.SecureProtocol
	AddConnectionHandler                  func(rvcid uint32, urls []string, ip string, port string)
	UpdateConnectionHandler               func(rvcid uint32, urls []string, ip string, port string)
	DoesConnectionExistHandler            func(rvcid uint32) bool
	ReplaceConnectionUrlHandler           func(rvcid uint32, oldurl string, newurl string)
)

// AddConnection sets the AddConnection handler function
func AddConnection(handler func(rvcid uint32, urls []string, ip string, port string)) {
	AddConnectionHandler = handler
}

// UpdateConnection sets the UpdateConnection handler function
func UpdateConnection(handler func(rvcid uint32, urls []string, ip string, port string)) {
	UpdateConnectionHandler = handler
}

// ReplaceConnectionUrl sets the ReplaceConnectionUrl handler function
func ReplaceConnectionUrl(handler func(rvcid uint32, oldurl string, newurl string)) {
	ReplaceConnectionUrlHandler = handler
}

// DoesConnectionExist sets the DoesConnectionExist handler function
func DoesConnectionExist(handler func(rvcid uint32) bool) {
	DoesConnectionExistHandler = handler
}

// InitSecureConnectionProtocol returns a new InitSecureConnectionProtocol
func InitSecureConnectionProtocol(nexServer *nex.Server) *nexproto.SecureProtocol {
	server = nexServer
	secureProtocolServer := nexproto.NewSecureProtocol(nexServer)

	secureProtocolServer.Register(register)
	secureProtocolServer.ReplaceURL(replaceURL)
	secureProtocolServer.SendReport(sendReport)
	return secureProtocolServer
}