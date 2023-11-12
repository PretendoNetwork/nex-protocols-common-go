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

	client := packet.Sender().(*nex.PRUDPClient)

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	server := commonMatchMakingProtocol.server
	hostPID := session.GameMatchmakeSession.Gathering.HostPID
	host := server.FindClientByPID(hostPID)
	if host == nil {
		common_globals.Logger.Warning("Host client not found, trying with owner client") // This popped up once during testing. Leaving it noted here in case it becomes a problem.
		host = server.FindClientByPID(session.GameMatchmakeSession.Gathering.OwnerPID)
		if host == nil {
			common_globals.Logger.Error("Owner client not found") // This popped up once during testing. Leaving it noted here in case it becomes a problem.
			return nex.Errors.Core.Exception
		}
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteListStationURL(host.StationURLs)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	server.Send(responsePacket)

	return 0
}
