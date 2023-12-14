package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSession(err error, packet nex.PacketInterface, callID uint32, gid uint32, strMessage string) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - Remove cast to PRUDPClient?
	client := packet.Sender().(*nex.PRUDPClient)

	server := commonProtocol.server

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - More checks here

	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID}, client, strMessage)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewStreamOut(server)

	if server.MatchMakingProtocolVersion().GreaterOrEqual("3.0.0") {
		rmcResponseStream.WriteBuffer(joinedMatchmakeSession.SessionKey)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodJoinMatchmakeSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
