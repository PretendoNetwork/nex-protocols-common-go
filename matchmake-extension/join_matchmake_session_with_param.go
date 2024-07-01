package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) joinMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, joinMatchmakeSessionParam *match_making_types.JoinMatchmakeSessionParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(joinMatchmakeSessionParam.JoinMessage.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	joinedMatchmakeSession, systemPassword, nexError := database.GetMatchmakeSessionByID(commonProtocol.db, endpoint, joinMatchmakeSessionParam.GID.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	// TODO - Are these the correct error codes?
	if joinedMatchmakeSession.UserPasswordEnabled.Value && !joinMatchmakeSessionParam.StrUserPassword.Equals(joinedMatchmakeSession.UserPassword) {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "change_error")
	}

	if joinedMatchmakeSession.SystemPasswordEnabled.Value && joinMatchmakeSessionParam.StrSystemPassword.Value != systemPassword {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPassword, "change_error")
	}

	nexError = common_globals.CanJoinMatchmakeSession(connection.PID(), joinedMatchmakeSession)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	_, nexError = match_making_database.JoinGatheringWithParticipants(commonProtocol.db, joinedMatchmakeSession.Gathering.ID.Value, connection, joinMatchmakeSessionParam.AdditionalParticipants.Slice(), joinMatchmakeSessionParam.JoinMessage.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	joinMatchmakeSessionParam.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterJoinMatchmakeSessionWithParam != nil {
		go commonProtocol.OnAfterJoinMatchmakeSessionWithParam(packet, joinMatchmakeSessionParam)
	}

	return rmcResponse, nil
}
