package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	//"os"
)

func getSessionURLs(err error, client *nex.Client, callID uint32, gatheringId uint32) {
	missingHandler := false
	if (GetConnectionUrlsHandler == nil){
		logger.Warning("MatchMaking::GetSessionURLs missing GetConnectionUrlsHandler!")
		missingHandler = true
	}
	if (GetRoomInfoHandler == nil){
		logger.Warning("MatchMaking::GetSessionURLs missing GetRoomInfoHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	var stationUrlStrings []string

	hostpid, _, _, _, _ := GetRoomInfoHandler(gatheringId)

	stationUrlStrings = GetConnectionUrlsHandler(server.FindClientFromPID(hostpid).ConnectionID())

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteListString(stationUrlStrings)

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchMakingMethodGetSessionURLs, rmcResponseBody)

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
