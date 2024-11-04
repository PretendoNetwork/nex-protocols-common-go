package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	database "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
)

func (commonProtocol *CommonProtocol) autoMatchmakePostpone(err error, packet nex.PacketInterface, callID uint32, anyGathering *types.AnyDataHolder, message *types.String) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.CleanupSearchMatchmakeSession == nil {
		common_globals.Logger.Warning("MatchmakeExtension::AutoMatchmake_Postpone missing CleanupSearchMatchmakeSession!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(message.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	database.EndMatchmakeSessionsParticipation(commonProtocol.manager, connection)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName

	if anyGatheringDataType.Value == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData.(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if !common_globals.CheckValidMatchmakeSession(matchmakeSession) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	searchMatchmakeSession := matchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	commonProtocol.CleanupSearchMatchmakeSession(searchMatchmakeSession)
	resultSession, nexError := database.FindMatchmakeSession(commonProtocol.manager, connection, searchMatchmakeSession)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	if resultSession == nil {
		resultSession = searchMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
		nexError = database.CreateMatchmakeSession(commonProtocol.manager, connection, resultSession)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	}

	participants, nexError := match_making_database.JoinGathering(commonProtocol.manager, resultSession.Gathering.ID.Value, connection, 1, message.Value)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	resultSession.ParticipationCount.Value = participants

	commonProtocol.manager.Mutex.Unlock()

	matchmakeDataHolder := types.NewAnyDataHolder()

	matchmakeDataHolder.TypeName = types.NewString("MatchmakeSession")
	matchmakeDataHolder.ObjectData = resultSession.Copy()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	matchmakeDataHolder.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakePostpone
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterAutoMatchmakePostpone != nil {
		go commonProtocol.OnAfterAutoMatchmakePostpone(packet, anyGathering, message)
	}

	return rmcResponse, nil
}
