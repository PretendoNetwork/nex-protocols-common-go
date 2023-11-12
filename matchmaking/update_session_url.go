package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func updateSessionURL(err error, packet nex.PacketInterface, callID uint32, idGathering uint32, strURL string) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	server := commonMatchMakingProtocol.server
	client := packet.Sender().(*nex.PRUDPClient)

	// * Mario Kart 7 seems to set an empty strURL, so I assume this is what the method does?
	session.GameMatchmakeSession.Gathering.HostPID = client.PID()
	if session.GameMatchmakeSession.Gathering.Flags&match_making.GatheringFlags.DisconnectChangeOwner != 0 {
		session.GameMatchmakeSession.Gathering.OwnerPID = client.PID()
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBool(true) // %retval%

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
