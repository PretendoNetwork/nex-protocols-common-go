package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server               nex.ServerInterface
	protocol             secure_connection.Interface
	CreateReportDBRecord func(pid *types.PID, reportID *types.PrimitiveU32, reportData *types.Buffer) error
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol secure_connection.Interface) *CommonProtocol {
	protocol.SetHandlerRegister(register)
	protocol.SetHandlerReplaceURL(replaceURL)
	protocol.SetHandlerSendReport(sendReport)

	commonProtocol = &CommonProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonProtocol
}
