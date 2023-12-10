package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func autoMatchmakeWithParam_Postpone(err error, packet nex.PacketInterface, callID uint32, autoMatchmakeParam *match_making_types.AutoMatchmakeParam) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonMatchmakeExtensionProtocol.server

	// TODO - Remove cast to PRUDPClient once websockets are implemented
	client := packet.Sender().(*nex.PRUDPClient)

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	common_globals.RemoveClientFromAllSessions(client)

	var matchmakeSession *match_making_types.MatchmakeSession
	matchmakeSession = autoMatchmakeParam.SourceMatchmakeSession

	sessions := common_globals.FindSessionsByMatchmakeSessionSearchCriterias(client.PID(), autoMatchmakeParam.LstSearchCriteria, commonMatchmakeExtensionProtocol.GameSpecificMatchmakeSessionSearchCriteriaChecks)
	var session *common_globals.CommonMatchmakeSession

	if len(sessions) == 0 {
		var errCode uint32
		session, err, errCode = common_globals.CreateSessionByMatchmakeSession(matchmakeSession, nil, client.PID())
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nil, errCode
		}
	} else {
		session = sessions[0]
	}

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID}, client, "")
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewStreamOut(server)
	matchmakeDataHolder := nex.NewDataHolder()
	matchmakeDataHolder.SetTypeName("MatchmakeSession")
	matchmakeDataHolder.SetObjectData(session.GameMatchmakeSession)
	rmcResponseStream.WriteStructure(session.GameMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakeWithParamPostpone
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
