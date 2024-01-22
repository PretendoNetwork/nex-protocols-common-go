package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, joinMatchmakeSessionParam *match_making_types.JoinMatchmakeSessionParam) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[joinMatchmakeSessionParam.GID.Value]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// TODO - More checks here
	err, errCode := common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, joinMatchmakeSessionParam.JoinMessage.Value)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	session.GameMatchmakeSession.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
