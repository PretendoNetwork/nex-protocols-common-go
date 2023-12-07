package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func sendReport(err error, packet nex.PacketInterface, callID uint32, reportID uint32, reportData []byte) (*nex.RMCMessage, uint32) {
	if commonSecureConnectionProtocol.createReportDBRecordHandler == nil {
		common_globals.Logger.Warning("SecureConnection::SendReport missing CreateReportDBRecord!")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Core.Unknown
	}

	client := packet.Sender()

	err = commonSecureConnectionProtocol.createReportDBRecordHandler(client.PID().LegacyValue(), reportID, reportData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Core.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodSendReport
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
