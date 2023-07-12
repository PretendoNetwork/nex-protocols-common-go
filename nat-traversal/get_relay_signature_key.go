package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func getRelaySignatureKey(err error, client *nex.Client, callID uint32) {
	server := commonNATTraversalProtocol.server
	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteInt32LE(0)
	dateTime := nex.NewDateTime(0)
	dateTime.UTC()
	rmcResponseStream.WriteDateTime(dateTime)
	rmcResponseStream.WriteString("") // Relay server address. We don't have one, so for now this is empty.
	rmcResponseStream.WriteUInt16LE(0) // Relay server port. We don't have one, so for now this is empty.
	rmcResponseStream.WriteInt32LE(0)
	rmcResponseStream.WriteUInt32LE(0) //Game Server ID. I don't know if this is checked (it doesn't appear to be though).
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nat_traversal.ProtocolID, callID)
	rmcResponse.SetSuccess(nat_traversal.MethodGetRelaySignatureKey, rmcResponseBody)

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
