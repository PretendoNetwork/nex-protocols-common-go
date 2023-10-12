package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSession(err error, client *nex.Client, callID uint32, anyGathering *nex.DataHolder, message string, participationCount uint16) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	// A client may disconnect from a session without leaving reliably,
	// so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName()

	if anyGatheringDataType == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData().(*match_making_types.MatchmakeSession)
	} else {
		logger.Critical("Non-MatchmakeSession DataType?!")
		return nex.Errors.Core.InvalidArgument
	}

	session, err, errCode := common_globals.CreateSessionBySearchCriteria(matchmakeSession, make([]*match_making_types.MatchmakeSessionSearchCriteria, 0), client.PID())
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client, message)
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteUInt32LE(session.GameMatchmakeSession.Gathering.ID)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		rmcResponseStream.WriteBuffer(session.GameMatchmakeSession.SessionKey)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodCreateMatchmakeSession, rmcResponseBody)

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
