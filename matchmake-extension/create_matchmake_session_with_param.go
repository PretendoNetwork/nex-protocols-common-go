package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSessionWithParam(err error, client *nex.Client, callID uint32, createMatchmakeSessionParam *match_making_types.CreateMatchmakeSessionParam) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	common_globals.RemoveClientFromAllSessions(client)

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	session, err, errCode := common_globals.CreateSessionByMatchmakeSession(joinedMatchmakeSession, nil, client.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client, createMatchmakeSessionParam.JoinMessage)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(session.GameMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodCreateMatchmakeSessionWithParam, rmcResponseBody)

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
