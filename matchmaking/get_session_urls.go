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
	if (GetRoomHandler == nil){
		logger.Warning("MatchMaking::GetSessionURLs missing GetRoomHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	var stationUrlStrings []string

	_, hostRVCID, _ := GetRoomHandler(gatheringId)

	hostClient := server.FindClientFromConnectionID(hostRVCID)
	if(hostClient != nil){
		stationUrlStrings = GetConnectionUrlsHandler(hostClient.ConnectionID())
	}else{
		rmcResponse := nex.NewRMCResponse(nexproto.MatchmakeExtensionProtocolID, callID)
		rmcResponse.SetError(0x8003006D)

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
	//stationUrlStrings[0] += ";R=1;Rsa=159.203.102.56;Rsp=9999;Ra=159.203.102.56;Rp=9999"
	//stationUrlStrings[1] += ";R=1;Rsa=159.203.102.56;Rsp=9999;Ra=159.203.102.56;Rp=9999"
	//logger.Info(stationUrlStrings[0])
	//logger.Info(stationUrlStrings[1])

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
