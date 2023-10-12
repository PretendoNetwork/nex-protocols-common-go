package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSession(err error, client *nex.Client, callID uint32, gid uint32, strMessage string) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	// TODO - More checks here

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client, strMessage)
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewStreamOut(server)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		rmcResponseStream.WriteBuffer(joinedMatchmakeSession.SessionKey)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodJoinMatchmakeSession, rmcResponseBody)

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
