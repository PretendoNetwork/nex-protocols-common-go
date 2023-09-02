package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func updateSessionURL(err error, client *nex.Client, callID uint32, idGathering uint32, strURL string) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	logger.Info("=== UpdateSessionURL ===")
	logger.Info(strURL)
	logger.Infof("PID: %d", client.PID())
	logger.Infof("CID: %d", client.ConnectionID())
	logger.Info("========================")

	server := commonMatchMakingProtocol.server
	hostPID := session.GameMatchmakeSession.Gathering.HostPID
	host := server.FindClientFromPID(hostPID)
	if host == nil {
		logger.Warning("Host client not found") // This popped up once during testing. Leaving it noted here in case it becomes a problem.
		return nex.Errors.Core.Exception
	}

	stations := host.StationURLs()

	stationURL := nex.NewStationURL(strURL)

	if stationURL.Type() == 3 {
		stations[1] = stationURL
	} else {
		stations[0] = stationURL
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBool(true) // %retval%

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

	return 0
}
