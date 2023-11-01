package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getRelaySignatureKey(err error, packet nex.PacketInterface, callID uint32) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender()
	server := commonNATTraversalProtocol.server

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteInt32LE(0)
	dateTime := nex.NewDateTime(0)
	dateTime.UTC()
	rmcResponseStream.WriteDateTime(dateTime)
	rmcResponseStream.WriteString("")  // Relay server address. We don't have one, so for now this is empty.
	rmcResponseStream.WriteUInt16LE(0) // Relay server port. We don't have one, so for now this is empty.
	rmcResponseStream.WriteInt32LE(0)
	rmcResponseStream.WriteUInt32LE(0) // Game Server ID. I don't know if this is checked (it doesn't appear to be though).
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
	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
