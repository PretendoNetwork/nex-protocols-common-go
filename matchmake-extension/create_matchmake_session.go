package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, anyGathering *nex.DataHolder, message string, participationCount uint16) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - Remove cast to PRUDPClient?
	client := packet.Sender().(*nex.PRUDPClient)

	server := commonProtocol.server

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName()

	if anyGatheringDataType == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData().(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, err, errCode := common_globals.CreateSessionByMatchmakeSession(matchmakeSession, nil, client.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID}, client, message)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteUInt32LE(session.GameMatchmakeSession.Gathering.ID)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		rmcResponseStream.WriteBuffer(session.GameMatchmakeSession.SessionKey)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
