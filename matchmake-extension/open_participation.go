package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func openParticipation(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.ResultCodes.RendezVous.SessionVoid
	}

	if session.GameMatchmakeSession.Gathering.OwnerPID.Equals(connection.PID()) {
		return nil, nex.ResultCodes.RendezVous.PermissionDenied
	}

	session.GameMatchmakeSession.OpenParticipation = types.NewPrimitiveBool(true)

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodOpenParticipation
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
