package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"
)

func (commonProtocol *CommonProtocol) joinMatchmakeSessionEx(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, strMessage *types.String, dontCareMyBlockList *types.PrimitiveBool, participationCount *types.PrimitiveU16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(strMessage.Value) > 256 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	common_globals.MatchmakingMutex.Lock()

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	joinedMatchmakeSession, nexError := database.GetMatchmakeSessionByID(commonProtocol.db, endpoint, gid.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	nexError = common_globals.CanJoinMatchmakeSession(connection.PID(), joinedMatchmakeSession)
	if nexError != nil {
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	_, nexError = match_making_database.JoinGathering(commonProtocol.db, joinedMatchmakeSession.Gathering.ID.Value, connection, participationCount.Value, strMessage.Value)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		common_globals.MatchmakingMutex.Unlock()
		return nil, nexError
	}

	common_globals.MatchmakingMutex.Unlock()

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	if server.LibraryVersions.MatchMaking.GreaterOrEqual("3.0.0") {
		joinedMatchmakeSession.SessionKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionEx
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterJoinMatchmakeSessionEx != nil {
		go commonProtocol.OnAfterJoinMatchmakeSessionEx(packet, gid, strMessage, dontCareMyBlockList, participationCount)
	}

	return rmcResponse, nil
}
