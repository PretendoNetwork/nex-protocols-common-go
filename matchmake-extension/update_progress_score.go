package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func updateProgressScore(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, progressScore *types.PrimitiveU8) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session := common_globals.Sessions[gid.Value]
	if session == nil {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	if progressScore.Value > 100 {
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	if session.GameMatchmakeSession.Gathering.OwnerPID.Equals(connection.PID()) {
		return nil, nex.Errors.RendezVous.PermissionDenied
	}

	session.GameMatchmakeSession.ProgressScore.Value += progressScore.Value

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateProgressScore
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
