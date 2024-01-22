package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, strMessage *types.String) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// TODO - More checks here
	err, errCode := common_globals.AddPlayersToSession(session, []uint32{connection.ID}, connection, strMessage.Value)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewByteStreamOut(server)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		joinedMatchmakeSession.SessionKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
