package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func autoMatchmake_Postpone(err error, packet nex.PacketInterface, callID uint32, anyGathering *types.AnyDataHolder, message *types.String) (*nex.RMCMessage, uint32) {
	if commonProtocol.CleanupSearchMatchmakeSession == nil {
		common_globals.Logger.Warning("MatchmakeExtension::AutoMatchmake_Postpone missing CleanupSearchMatchmakeSession!")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from the session
	common_globals.RemoveConnectionFromAllSessions(connection)

	var matchmakeSession *match_making_types.MatchmakeSession
	anyGatheringDataType := anyGathering.TypeName

	if anyGatheringDataType.Value == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData.(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		return nil, nex.Errors.Core.InvalidArgument
	}

	searchMatchmakeSession := matchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	commonProtocol.CleanupSearchMatchmakeSession(searchMatchmakeSession)
	sessionIndex := common_globals.FindSessionByMatchmakeSession(connection.PID(), searchMatchmakeSession)
	var session *common_globals.CommonMatchmakeSession

	if sessionIndex == 0 {
		var errCode uint32
		session, err, errCode = common_globals.CreateSessionByMatchmakeSession(matchmakeSession, searchMatchmakeSession, connection.PID())
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nil, errCode
		}
	} else {
		session = common_globals.Sessions[sessionIndex]
	}

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, message.Value)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	matchmakeDataHolder := types.NewAnyDataHolder()

	matchmakeDataHolder.TypeName = types.NewString("MatchmakeSession")
	matchmakeDataHolder.ObjectData = session.GameMatchmakeSession.Copy()

	rmcResponseStream := nex.NewByteStreamOut(server)

	matchmakeDataHolder.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodAutoMatchmakePostpone
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
