package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func updateSessionHostV1(err error, client *nex.Client, callID uint32, gid uint32) {
	missingHandler := false
	if (UpdateRoomHostHandler == nil){
		logger.Warning("MatchMaking::UpdateSessionHostV1 missing UpdateRoomHostHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	UpdateRoomHostHandler(gid, client.ConnectionID(), client.PID())

	rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchMakingMethodUpdateSessionHostV1, nil)

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