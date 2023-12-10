package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

var commonSecureConnectionProtocol *CommonSecureConnectionProtocol

type CommonSecureConnectionProtocol struct {
	server               nex.ServerInterface
	protocol             secure_connection.Interface
	CreateReportDBRecord func(pid uint32, reportID uint32, reportData []byte) error
}

// NewCommonSecureConnectionProtocol returns a new CommonSecureConnectionProtocol
func NewCommonSecureConnectionProtocol(protocol secure_connection.Interface) *CommonSecureConnectionProtocol {
	protocol.SetHandlerRegister(register)
	protocol.SetHandlerReplaceURL(replaceURL)
	protocol.SetHandlerSendReport(sendReport)

	commonSecureConnectionProtocol = &CommonSecureConnectionProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonSecureConnectionProtocol
}
