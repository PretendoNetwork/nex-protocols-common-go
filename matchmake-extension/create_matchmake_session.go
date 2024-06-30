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

func (commonProtocol *CommonProtocol) createMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, anyGathering *types.AnyDataHolder, message *types.String, participationCount *types.PrimitiveU16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	common_globals.MatchmakingMutex.Lock()

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	database.EndMatchmakeSessionsParticipation(commonProtocol.db, connection)

	var matchmakeSession *match_making_types.MatchmakeSession

	if anyGathering.TypeName.Value == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData.(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		common_globals.MatchmakingMutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	nexError := database.CreateMatchmakeSession(commonProtocol.db, connection, matchmakeSession)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	participants, nexError := match_making_database.JoinGathering(commonProtocol.db, matchmakeSession.Gathering.ID.Value, connection, participationCount.Value, message.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	matchmakeSession.ParticipationCount.Value = participants

	common_globals.MatchmakingMutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	matchmakeSession.Gathering.ID.WriteTo(rmcResponseStream)

	if server.LibraryVersions.MatchMaking.GreaterOrEqual("3.0.0") {
		matchmakeSession.SessionKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSession
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCreateMatchmakeSession != nil {
		go commonProtocol.OnAfterCreateMatchmakeSession(packet, anyGathering, message, participationCount)
	}

	return rmcResponse, nil
}
