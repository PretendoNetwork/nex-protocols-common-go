package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

type CommonProtocol struct {
	endpoint             nex.EndpointInterface
	protocol             secure_connection.Interface
	CreateReportDBRecord func(pid *types.PID, reportID *types.PrimitiveU32, reportData *types.QBuffer) error
	OnAfterRegister      func(packet nex.PacketInterface, vecMyURLs *types.List[*types.StationURL])
	OnAfterReplaceURL    func(packet nex.PacketInterface, target *types.StationURL, url *types.StationURL)
	OnAfterSendReport    func(packet nex.PacketInterface, reportID *types.PrimitiveU32, reportData *types.QBuffer)
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol secure_connection.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerRegister(commonProtocol.register)
	protocol.SetHandlerReplaceURL(commonProtocol.replaceURL)
	protocol.SetHandlerSendReport(commonProtocol.sendReport)

	return commonProtocol
}
