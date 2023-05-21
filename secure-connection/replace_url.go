package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

func replaceURL(err error, client *nex.Client, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) {
	server := commonSecureConnectionProtocol.server
	if commonSecureConnectionProtocol.replaceConnectionUrlHandler == nil {
		logger.Warning("Missing ReplaceConnectionUrlHandler!")
		return
	}

	urls := client.StationURLs()

	for i := 0; i < len(urls); i++ {
		if urls[i] == oldStation.EncodeToString() {
			urls[i] = newStation.EncodeToString()
		}
	}
	client.SetStationURLs(urls)

	rmcResponse := nex.NewRMCResponse(secure_connection.ProtocolID, callID)
	rmcResponse.SetSuccess(secure_connection.MethodReplaceURL, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
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

	server.Send(responsePacket)
}
