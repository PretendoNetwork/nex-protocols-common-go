package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func createMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, createMatchmakeSessionParam *match_making_types.CreateMatchmakeSessionParam) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// * A client may disconnect from a session without leaving reliably,
	// * so let's make sure the client is removed from all sessions
	common_globals.RemoveConnectionFromAllSessions(connection)

	joinedMatchmakeSession := createMatchmakeSessionParam.SourceMatchmakeSession.Copy().(*match_making_types.MatchmakeSession)
	session, err, errCode := common_globals.CreateSessionByMatchmakeSession(joinedMatchmakeSession, nil, connection.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	err, errCode = common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, createMatchmakeSessionParam.JoinMessage.Value)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	session.GameMatchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodCreateMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
