package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func sendReport(err error, packet nex.PacketInterface, callID uint32, reportID *types.PrimitiveU32, reportData *types.Buffer) (*nex.RMCMessage, uint32) {
	if commonProtocol.CreateReportDBRecord == nil {
		common_globals.Logger.Warning("SecureConnection::SendReport missing CreateReportDBRecord!")
		return nil, nex.ResultCodesCore.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.ResultCodesCore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	err = commonProtocol.CreateReportDBRecord(connection.PID(), reportID, reportData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.ResultCodesCore.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodSendReport
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
