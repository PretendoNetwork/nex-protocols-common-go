package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func acquireNexUniqueID(err error, packet nex.PacketInterface, callID uint32) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender()

	var pNexUniqueID uint64

	if commonUtilityProtocol.randomU64Handler != nil {
		pNexUniqueID = commonUtilityProtocol.randomU64Handler()
	} else {
		pNexUniqueID = commonUtilityProtocol.randGenerator.Uint64()
	}

	server := commonUtilityProtocol.server

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteUInt64LE(pNexUniqueID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(utility.ProtocolID, callID)
	rmcResponse.SetSuccess(utility.MethodAcquireNexUniqueID, rmcResponseBody)

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
