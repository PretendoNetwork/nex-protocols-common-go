package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, createMatchmakeSessionParam *match_making_types.CreateMatchmakeSessionParam) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - Remove cast to PRUDPClient?
	client := packet.Sender().(*nex.PRUDPClient)

	server := commonMatchmakeExtensionProtocol.server

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	common_globals.RemoveClientFromAllSessions(client)

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	session, err, errCode := common_globals.CreateSessionByMatchmakeSession(joinedMatchmakeSession, nil, client.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID}, client, createMatchmakeSessionParam.JoinMessage)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(session.GameMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
