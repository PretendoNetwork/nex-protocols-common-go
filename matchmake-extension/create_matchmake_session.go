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
	common_globals.RemoveConnectionIDFromAllSessions(client.ConnectionID())

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName()

	if anyGatheringDataType == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData().(*match_making_types.MatchmakeSession)
	} else {
		logger.Critical("Non-MatchmakeSession DataType?!")
		return nex.Errors.Core.InvalidArgument
	}

	sessionIndex := common_globals.GetSessionIndex()
	// This should in theory be impossible, as there aren't enough PIDs creating sessions to fill the uint32 limit.
	// If we ever get here, we must be not deleting sessions properly
	if sessionIndex == 0 {
		logger.Critical("No gatherings available!")
		return nex.Errors.RendezVous.LimitExceeded
	}

	session := common_globals.CommonMatchmakeSession{
		SearchCriteria:       make([]*match_making_types.MatchmakeSessionSearchCriteria, 0),
		GameMatchmakeSession: matchmakeSession,
	}

	common_globals.Sessions[sessionIndex] = &session
	common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.ID = sessionIndex
	common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.OwnerPID = client.PID()
	common_globals.Sessions[sessionIndex].GameMatchmakeSession.Gathering.HostPID = client.PID()

	common_globals.Sessions[sessionIndex].GameMatchmakeSession.StartedTime = nex.NewDateTime(0)
	common_globals.Sessions[sessionIndex].GameMatchmakeSession.StartedTime.UTC()
	common_globals.Sessions[sessionIndex].GameMatchmakeSession.SessionKey = make([]byte, 32)

	err, errCode := common_globals.AddPlayersToSession(common_globals.Sessions[sessionIndex], []uint32{client.ConnectionID()})
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteUInt32LE(sessionIndex)

	if server.MatchMakingProtocolVersion().Major <= 3 {
		rmcResponseStream.WriteBuffer(matchmakeSession.SessionKey)
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
