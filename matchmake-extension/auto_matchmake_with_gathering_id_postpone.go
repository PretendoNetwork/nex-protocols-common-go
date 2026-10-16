package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) autoMatchmakeWithGatheringIDPostpone(err error, packet nex.PacketInterface, callID uint32, lstGID types.List[types.UInt32], anyGathering match_making_types.GatheringHolder, strMessage types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	if len(strMessage) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol.manager.Mutex.Lock()

	var matchmakeSession match_making_types.MatchmakeSession

	if anyGathering.Object.GatheringObjectID().Equals(types.NewString("MatchmakeSession")) {
		matchmakeSession = anyGathering.Object.(match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if !common_globals.CheckValidMatchmakeSession(matchmakeSession) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	resultSessions, nexError := database.FindMatchmakeSessionsByID(commonProtocol.manager, endpoint, lstGID)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	var resultSession match_making_types.MatchmakeSession
	if len(resultSessions) == 0 {
		resultSession = anyGathering.Object.Copy().(match_making_types.MatchmakeSession)
		nexError = database.CreateMatchmakeSession(commonProtocol.manager, connection, &resultSession)
		if nexError != nil {
			common_globals.Logger.Error(nexError.Error())
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	} else {
		resultSession = resultSessions[0]
	}

	participants, nexError := database.JoinMatchmakeSession(commonProtocol.manager, resultSession, connection, 1, string(strMessage))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	resultSession.ParticipationCount = types.NewUInt32(participants)

	commonProtocol.manager.Mutex.Unlock()

	matchmakeDataHolder := match_making_types.NewGatheringHolder()
	matchmakeDataHolder.Object = resultSession.Copy().(match_making_types.GatheringInterface)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	matchmakeDataHolder.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakeWithGatheringIDPostpone
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterAutoMatchmakeWithParamPostpone != nil {
		go commonProtocol.OnAfterAutoMatchmakeWithGatheringIDPostpone(packet, lstGID)
	}

	return rmcResponse, nil
}
