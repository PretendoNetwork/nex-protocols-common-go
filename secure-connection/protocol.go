package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"github.com/PretendoNetwork/plogger-go"
)

var commonSecureConnectionProtocol *CommonSecureConnectionProtocol
var logger = plogger.NewLogger()

type CommonSecureConnectionProtocol struct {
	*nexproto.SecureProtocol
	server *nex.Server

	addConnectionHandler        func(rvcid uint32, urls []string, ip string, port string)
	updateConnectionHandler     func(rvcid uint32, urls []string, ip string, port string)
	doesConnectionExistHandler  func(rvcid uint32) bool
	replaceConnectionUrlHandler func(rvcid uint32, oldurl string, newurl string)
}

// AddConnection sets the AddConnection handler function
func (commonSecureConnectionProtocol *CommonSecureConnectionProtocol) AddConnection(handler func(rvcid uint32, urls []string, ip string, port string)) {
	commonSecureConnectionProtocol.addConnectionHandler = handler
}

// UpdateConnection sets the UpdateConnection handler function
func (commonSecureConnectionProtocol *CommonSecureConnectionProtocol) UpdateConnection(handler func(rvcid uint32, urls []string, ip string, port string)) {
	commonSecureConnectionProtocol.updateConnectionHandler = handler
}

// ReplaceConnectionUrl sets the ReplaceConnectionUrl handler function
func (commonSecureConnectionProtocol *CommonSecureConnectionProtocol) ReplaceConnectionUrl(handler func(rvcid uint32, oldurl string, newurl string)) {
	commonSecureConnectionProtocol.replaceConnectionUrlHandler = handler
}

// DoesConnectionExist sets the DoesConnectionExist handler function
func (commonSecureConnectionProtocol *CommonSecureConnectionProtocol) DoesConnectionExist(handler func(rvcid uint32) bool) {
	commonSecureConnectionProtocol.doesConnectionExistHandler = handler
}

// NewCommonSecureConnectionProtocol returns a new CommonSecureConnectionProtocol
func NewCommonSecureConnectionProtocol(server *nex.Server) *CommonSecureConnectionProtocol {
	secureProtocol := nexproto.NewSecureProtocol(server)
	commonSecureConnectionProtocol = &CommonSecureConnectionProtocol{SecureProtocol: secureProtocol, server: server}

	server.On("Connect", connect)
	commonSecureConnectionProtocol.Register(register)
	commonSecureConnectionProtocol.ReplaceURL(replaceURL)
	commonSecureConnectionProtocol.SendReport(sendReport)

	return commonSecureConnectionProtocol
}
