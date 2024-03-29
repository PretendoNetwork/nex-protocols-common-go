package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func getSessionURLs(err error, packet nex.PacketInterface, callID uint32, gid uint32) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender()

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	server := commonMatchMakingProtocol.server
	hostPID := session.GameMatchmakeSession.Gathering.HostPID
	host := server.FindClientFromPID(hostPID)
	if host == nil {
		common_globals.Logger.Warning("Host client not found, trying with owner client") // This popped up once during testing. Leaving it noted here in case it becomes a problem.
		host = server.FindClientFromPID(session.GameMatchmakeSession.Gathering.OwnerPID)
		if host == nil {
			common_globals.Logger.Error("Owner client not found") // This popped up once during testing. Leaving it noted here in case it becomes a problem.
			return nex.Errors.Core.Exception
		}
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteListStationURL(host.StationURLs())

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

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
