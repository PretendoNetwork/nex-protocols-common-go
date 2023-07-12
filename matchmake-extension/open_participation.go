package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func openParticipation(err error, client *nex.Client, callID uint32, gid uint32) {
	server := commonMatchmakeExtensionProtocol.server
	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	if session, ok := common_globals.Sessions[gid]; ok{
		session.GameMatchmakeSession.OpenParticipation = true
		rmcResponse.SetSuccess(matchmake_extension.MethodOpenParticipation, nil)
	} else {
		rmcResponse.SetError(nex.Errors.RendezVous.SessionVoid)
	}

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
}