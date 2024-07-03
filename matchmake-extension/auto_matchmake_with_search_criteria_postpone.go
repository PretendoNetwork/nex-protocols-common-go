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

func (commonProtocol *CommonProtocol) autoMatchmakeWithSearchCriteriaPostpone(err error, packet nex.PacketInterface, callID uint32, lstSearchCriteria *types.List[*match_making_types.MatchmakeSessionSearchCriteria], anyGathering *types.AnyDataHolder, strMessage *types.String) (*nex.RMCMessage, *nex.Error) {
	if commonProtocol.CleanupMatchmakeSessionSearchCriterias == nil {
		common_globals.Logger.Warning("MatchmakeExtension::AutoMatchmakeWithSearchCriteria_Postpone missing CleanupMatchmakeSessionSearchCriterias!")
		return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(strMessage.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	database.EndMatchmakeSessionsParticipation(commonProtocol.manager, connection)

	var matchmakeSession *match_making_types.MatchmakeSession

	if anyGathering.TypeName.Value == "MatchmakeSession" {
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

	commonProtocol.CleanupMatchmakeSessionSearchCriterias(lstSearchCriteria)

	resultRange := types.NewResultRange()
	resultRange.Length.Value = 1
	resultSessions, nexError := database.FindMatchmakeSessionBySearchCriteria(commonProtocol.manager, connection, lstSearchCriteria.Slice(), resultRange)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	var resultSession *match_making_types.MatchmakeSession
	if len(resultSessions) == 0 {
		resultSession = matchmakeSession.Copy().(*match_making_types.MatchmakeSession)
		nexError = database.CreateMatchmakeSession(commonProtocol.manager, connection, resultSession)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	} else {
		resultSession = resultSessions[0]

		// TODO - What should really happen here?
		if resultSession.UserPasswordEnabled.Value || resultSession.SystemPasswordEnabled.Value {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
		}
	}

	var vacantParticipants uint16 = 1
	if searchCriteria, err := lstSearchCriteria.Get(0); err == nil {
		vacantParticipants = searchCriteria.VacantParticipants.Value
	}

	participants, nexError := match_making_database.JoinGathering(commonProtocol.manager, resultSession.Gathering.ID.Value, connection, vacantParticipants, strMessage.Value)
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
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakeWithSearchCriteriaPostpone
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterAutoMatchmakeWithSearchCriteriaPostpone != nil {
		go commonProtocol.OnAfterAutoMatchmakeWithSearchCriteriaPostpone(packet, lstSearchCriteria, anyGathering, strMessage)
	}

	return rmcResponse, nil
}
