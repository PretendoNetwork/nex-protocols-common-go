package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, anyGathering *types.AnyDataHolder, message *types.String, participationCount *types.PrimitiveU16) (*nex.RMCMessage, uint32) {
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

	if anyGathering.TypeName.Value == "MatchmakeSession" {
		matchmakeSession = anyGathering.ObjectData.(*match_making_types.MatchmakeSession)
	} else {
		common_globals.Logger.Critical("Non-MatchmakeSession DataType?!")
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, err, errCode := common_globals.CreateSessionByMatchmakeSession(matchmakeSession, nil, connection.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, message.Value)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	session.GameMatchmakeSession.Gathering.ID.WriteTo(rmcResponseStream)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		session.GameMatchmakeSession.SessionKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
