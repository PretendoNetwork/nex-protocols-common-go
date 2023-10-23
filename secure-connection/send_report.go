package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func sendReport(err error, client *nex.Client, callID uint32, reportID uint32, reportData []byte) uint32 {
	if commonSecureConnectionProtocol.createReportDBRecordHandler == nil {
		common_globals.Logger.Warning("SecureConnection::SendReport missing CreateReportDBRecord!")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nex.Errors.Core.Unknown
	}

	err = commonSecureConnectionProtocol.createReportDBRecordHandler(client.PID(), reportID, reportData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nex.Errors.Core.Unknown
	}

	rmcResponse := nex.NewRMCResponse(secure_connection.ProtocolID, callID)
	rmcResponse.SetSuccess(secure_connection.MethodSendReport, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonSecureConnectionProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonSecureConnectionProtocol.server.Send(responsePacket)

	return 0
}
