package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

var commonSecureConnectionProtocol *CommonSecureConnectionProtocol

type CommonSecureConnectionProtocol struct {
	*secure_connection.Protocol
	server               *nex.PRUDPServer
	CreateReportDBRecord func(pid uint32, reportID uint32, reportData []byte) error
}

// NewCommonSecureConnectionProtocol returns a new CommonSecureConnectionProtocol
func NewCommonSecureConnectionProtocol(server *nex.PRUDPServer) *CommonSecureConnectionProtocol {
	secureConnectionProtocol := secure_connection.NewProtocol(server)
	commonSecureConnectionProtocol = &CommonSecureConnectionProtocol{Protocol: secureConnectionProtocol, server: server}

	commonSecureConnectionProtocol.Register = register
	commonSecureConnectionProtocol.ReplaceURL = replaceURL
	commonSecureConnectionProtocol.SendReport = sendReport

	return commonSecureConnectionProtocol
}
