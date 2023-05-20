package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func unregisterGathering(err error, client *nex.Client, callID uint32, gatheringId uint32) {
	server := commonMatchMakingProtocol.server
	missingHandler := false
	if commonMatchMakingProtocol.DestroyRoomHandler == nil {
		logger.Warning("MatchMaking::UnregisterGathering missing DestroyRoomHandler!")
		missingHandler = true
	}
	if missingHandler {
		return
	}
	commonMatchMakingProtocol.DestroyRoomHandler(gatheringId)
	rmcResponse := nex.NewRMCResponse(match_making.ProtocolID, callID)
	rmcResponse.SetSuccess(match_making.MethodUnregisterGathering, nil)

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
