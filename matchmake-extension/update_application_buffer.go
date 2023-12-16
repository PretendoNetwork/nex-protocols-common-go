package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func updateApplicationBuffer(err error, packet nex.PacketInterface, callID uint32, gid uint32, applicationBuffer []byte) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonProtocol.server

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - Should ANYONE be allowed to do this??

	session.GameMatchmakeSession.ApplicationData = applicationBuffer

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateApplicationBuffer
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
