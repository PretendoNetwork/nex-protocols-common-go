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

	server := commonMatchMakingProtocol.server

	// * Mario Kart 7 seems to set an empty strURL, so I assume this is what the method does?
	session.GameMatchmakeSession.Gathering.HostPID = client.PID()
	if session.GameMatchmakeSession.Gathering.Flags&match_making.GatheringFlags.DisconnectChangeOwner != 0 {
		session.GameMatchmakeSession.Gathering.OwnerPID = client.PID()
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
