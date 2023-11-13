package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func sendReport(err error, packet nex.PacketInterface, callID uint32, reportID uint32, reportData []byte) uint32 {
	if commonSecureConnectionProtocol.createReportDBRecordHandler == nil {
		common_globals.Logger.Warning("SecureConnection::SendReport missing CreateReportDBRecord!")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nex.Errors.Core.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	err = commonSecureConnectionProtocol.createReportDBRecordHandler(client.PID().LegacyValue(), reportID, reportData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nex.Errors.Core.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodSendReport
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if commonSecureConnectionProtocol.server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	commonSecureConnectionProtocol.server.Send(responsePacket)

	return 0
}
