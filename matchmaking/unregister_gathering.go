package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func unregisterGathering(err error, client *nex.Client, callID uint32, gatheringId uint32) {
	missingHandler := false
	if (DestroyRoomHandler == nil){
		logger.Warning("MatchMaking::UnregisterGathering missing DestroyRoomHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	DestroyRoomHandler(gatheringId)
	rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchMakingMethodUnregisterGathering, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if(server.PrudpVersion() == 0){
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	}else{
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
