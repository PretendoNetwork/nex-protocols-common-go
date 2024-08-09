package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) createMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, createMatchmakeSessionParam *match_making_types.CreateMatchmakeSessionParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if !common_globals.CheckValidMatchmakeSession(createMatchmakeSessionParam.SourceMatchmakeSession) {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(createMatchmakeSessionParam.JoinMessage.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	database.EndMatchmakeSessionsParticipation(commonProtocol.manager, connection)

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	nexError := database.CreateMatchmakeSession(commonProtocol.manager, connection, joinedMatchmakeSession)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	participants, nexError := match_making_database.JoinGatheringWithParticipants(commonProtocol.manager, joinedMatchmakeSession.Gathering.ID.Value, connection, createMatchmakeSessionParam.AdditionalParticipants.Slice(), createMatchmakeSessionParam.JoinMessage.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	joinedMatchmakeSession.ParticipationCount.Value = participants

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	joinedMatchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCreateMatchmakeSessionWithParam != nil {
		go commonProtocol.OnAfterCreateMatchmakeSessionWithParam(packet, createMatchmakeSessionParam)
	}

	return rmcResponse, nil
}
