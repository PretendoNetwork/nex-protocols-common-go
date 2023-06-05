package nattraversal

import (
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func requestProbeInitiationExt(err error, client *nex.Client, callID uint32, targetList []string, stationToProbe string) {
	server := commonNATTraversalProtocol.server
	rmcResponse := nex.NewRMCResponse(nat_traversal.ProtocolID, callID)
	rmcResponse.SetSuccess(nat_traversal.MethodRequestProbeInitiationExt, nil)

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

	rmcMessage := nex.NewRMCRequest()
	rmcMessage.SetProtocolID(nat_traversal.ProtocolID)
	rmcMessage.SetCallID(0xffff0000 + callID)
	rmcMessage.SetMethodID(nat_traversal.MethodInitiateProbe)

	rmcRequestStream := nex.NewStreamOut(server)
	rmcRequestStream.WriteString(stationToProbe)
	rmcRequestBody := rmcRequestStream.Bytes()

	rmcMessage.SetParameters(rmcRequestBody)
	rmcMessageBytes := rmcMessage.Bytes()

	for _, target := range targetList {
		targetUrl := nex.NewStationURL(target)
		logger.Info("Target: " + target)
		logger.Info("ToProbe: " + stationToProbe)
		targetRVCID, _ := strconv.Atoi(targetUrl.RVCID())
		targetClient := server.FindClientFromConnectionID(uint32(targetRVCID))
		if targetClient != nil {
			var messagePacket nex.PacketInterface

			if server.PRUDPVersion() == 0 {
				messagePacket, _ = nex.NewPacketV0(targetClient, nil)
				messagePacket.SetVersion(0)
			} else {
				messagePacket, _ = nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
			}

			messagePacket.SetSource(0xA1)
			messagePacket.SetDestination(0xAF)
			messagePacket.SetType(nex.DataPacket)
			messagePacket.SetPayload(rmcMessageBytes)

			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			server.Send(messagePacket)
		} else {
			logger.Warning("Client not found")
		}
	}
}
