package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSessionWithParam(err error, packet nex.PacketInterface, callID uint32, joinMatchmakeSessionParam *match_making_types.JoinMatchmakeSessionParam) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonMatchmakeExtensionProtocol.server

	session, ok := common_globals.Sessions[joinMatchmakeSessionParam.GID]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - More checks here
	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID}, client, joinMatchmakeSessionParam.JoinMessage)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(joinedMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSessionWithParam
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
