package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) autoMatchmakeWithParamPostpone(err error, packet nex.PacketInterface, callID uint32, autoMatchmakeParam *match_making_types.AutoMatchmakeParam) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.CleanupMatchmakeSessionSearchCriterias == nil {
		common_globals.Logger.Warning("MatchmakeExtension::AutoMatchmakeWithParam_Postpone missing CleanupMatchmakeSessionSearchCriterias!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if !common_globals.CheckValidMatchmakeSession(autoMatchmakeParam.SourceMatchmakeSession) {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(autoMatchmakeParam.JoinMessage.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	common_globals.MatchmakingMutex.Lock()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	database.EndMatchmakeSessionsParticipation(commonProtocol.db, connection)

	commonProtocol.CleanupMatchmakeSessionSearchCriterias(autoMatchmakeParam.LstSearchCriteria)

	resultRange := types.NewResultRange()
	resultRange.Length.Value = 1
	resultSessions, nexError := database.FindMatchmakeSessionBySearchCriteria(commonProtocol.db, connection, autoMatchmakeParam.LstSearchCriteria.Slice(), resultRange)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	var resultSession *match_making_types.MatchmakeSession
	if len(resultSessions) == 0 {
		resultSession = autoMatchmakeParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
		nexError = database.CreateMatchmakeSession(commonProtocol.db, connection, resultSession)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			common_globals.MatchmakingMutex.Unlock()
			return nil, nexError
		}
	} else {
		resultSession = resultSessions[0]
	}

	participants, nexError := match_making_database.JoinGatheringWithParticipants(commonProtocol.db, resultSession.ID.Value, connection, autoMatchmakeParam.AdditionalParticipants.Slice(), autoMatchmakeParam.JoinMessage.Value)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	resultSession.ParticipationCount.Value = participants

	common_globals.MatchmakingMutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	resultSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakeWithParamPostpone
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterAutoMatchmakeWithParamPostpone != nil {
		go commonProtocol.OnAfterAutoMatchmakeWithParamPostpone(packet, autoMatchmakeParam)
	}

	return rmcResponse, nil
}
