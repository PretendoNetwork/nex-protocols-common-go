package utility

import (
	nex "github.com/PretendoNetwork/nex-go"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

func acquireNexUniqueID(err error, client *nex.Client, callID uint32) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

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

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
