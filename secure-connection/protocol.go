package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
	"github.com/PretendoNetwork/plogger-go"
)

var commonSecureConnectionProtocol *CommonSecureConnectionProtocol
var logger = plogger.NewLogger()

type CommonSecureConnectionProtocol struct {
	*secure_connection.Protocol
	server *nex.Server
}

// NewCommonSecureConnectionProtocol returns a new CommonSecureConnectionProtocol
func NewCommonSecureConnectionProtocol(server *nex.Server) *CommonSecureConnectionProtocol {
	secureConnectionProtocol := secure_connection.NewProtocol(server)
	commonSecureConnectionProtocol = &CommonSecureConnectionProtocol{Protocol: secureConnectionProtocol, server: server}

	server.On("Connect", connect)
	commonSecureConnectionProtocol.Register(register)
	commonSecureConnectionProtocol.ReplaceURL(replaceURL)
	commonSecureConnectionProtocol.SendReport(sendReport)

	return commonSecureConnectionProtocol
}
