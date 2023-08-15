package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func updateProgressScore(err error, client *nex.Client, callID uint32, gid uint32, progressScore uint8) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	session := common_globals.Sessions[gid]
	if session == nil {
		return nex.Errors.RendezVous.SessionVoid
	}

	if progressScore > 100 {
		return nex.Errors.Core.InvalidArgument
	}

	if session.GameMatchmakeSession.Gathering.OwnerPID != client.PID() {
		return nex.Errors.RendezVous.PermissionDenied
	}

	session.GameMatchmakeSession.ProgressScore += progressScore

	server := commonMatchmakeExtensionProtocol.server

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodUpdateProgressScore, nil)

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
