package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func updateProgressScore(err error, packet nex.PacketInterface, callID uint32, gid uint32, progressScore uint8) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender()

	session := common_globals.Sessions[gid]
	if session == nil {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	if progressScore > 100 {
		return nil, nex.Errors.Core.InvalidArgument
	}

	if session.GameMatchmakeSession.Gathering.OwnerPID.Equals(client.PID()) {
		return nil, nex.Errors.RendezVous.PermissionDenied
	}

	session.GameMatchmakeSession.ProgressScore += progressScore

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateProgressScore
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
