package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func requestProbeInitiationExt(err error, packet nex.PacketInterface, callID uint32, targetList []string, stationToProbe string) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonNATTraversalProtocol.server

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodRequestProbeInitiationExt
	rmcResponse.CallID = callID

	rmcRequestStream := nex.NewStreamOut(server)
	rmcRequestStream.WriteString(stationToProbe)

	rmcRequestBody := rmcRequestStream.Bytes()

	rmcRequest := nex.NewRMCRequest()
	rmcRequest.ProtocolID = nat_traversal.ProtocolID
	rmcRequest.CallID = 0xffff0000 + callID
	rmcRequest.MethodID = nat_traversal.MethodInitiateProbe
	rmcRequest.Parameters = rmcRequestBody

	rmcRequestBytes := rmcRequest.Bytes()

	for _, target := range targetList {
		targetStation := nex.NewStationURL(target)
		targetClient := server.FindClientByConnectionID(targetStation.RVCID())
		if targetClient != nil {
			var messagePacket nex.PRUDPPacketInterface

			if server.PRUDPVersion == 0 {
				messagePacket, _ = nex.NewPRUDPPacketV0(targetClient, nil)
			} else {
				messagePacket, _ = nex.NewPRUDPPacketV1(targetClient, nil)
			}

			messagePacket.SetType(nex.DataPacket)
			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)
			messagePacket.SetSourceStreamType(client.DestinationStreamType)
			messagePacket.SetSourcePort(client.DestinationPort)
			messagePacket.SetDestinationStreamType(client.SourceStreamType)
			messagePacket.SetDestinationPort(client.SourcePort)
			messagePacket.SetPayload(rmcRequestBytes)

			server.Send(messagePacket)
		} else {
			common_globals.Logger.Warning("Client not found")
		}
	}

	return rmcResponse, 0
}
