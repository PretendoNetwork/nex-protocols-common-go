package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	"fmt"
)

func getSessionURLs(err error, client *nex.Client, callID uint32, gatheringId uint32) {
	server := commonMatchMakingProtocol.server
	missingHandler := false
	if commonMatchMakingProtocol.GetConnectionUrlsHandler == nil {
		logger.Warning("MatchMaking::GetSessionURLs missing GetConnectionUrlsHandler!")
		missingHandler = true
	}
	if commonMatchMakingProtocol.GetRoomInfoHandler == nil {
		logger.Warning("MatchMaking::GetSessionURLs missing GetRoomInfoHandler!")
		missingHandler = true
	}
	if missingHandler {
		return
	}
	var stationUrlStrings []string

	hostpid := common_globals.Sessions[gatheringId].GameMatchmakeSession.Gathering.HostPID

	hostclient := server.FindClientFromPID(hostpid)
	if(hostclient == nil){
		fmt.Println("nil hostclient?!") //this popped up once during testing. Leaving it noted here in case it becomes a problem.
		return
	}
	stationUrlStrings = hostclient.StationURLs()

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteListString(stationUrlStrings)

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(match_making.ProtocolID, callID)
	rmcResponse.SetSuccess(match_making.MethodGetSessionURLs, rmcResponseBody)

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
