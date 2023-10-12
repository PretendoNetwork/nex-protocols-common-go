package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func joinMatchmakeSessionWithParam(err error, client *nex.Client, callID uint32, joinMatchmakeSessionParam *match_making_types.JoinMatchmakeSessionParam) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := client.Server()

	session, ok := common_globals.Sessions[joinMatchmakeSessionParam.GID]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	// TODO - More checks here
	err, errCode := common_globals.AddPlayersToSession(session, []uint32{client.ConnectionID()}, client, "")
	if err != nil {
		logger.Error(err.Error())
		return errCode
	}

	joinedMatchmakeSession := session.GameMatchmakeSession

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(joinedMatchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodJoinMatchmakeSessionWithParam, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
