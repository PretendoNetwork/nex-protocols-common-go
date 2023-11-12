package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func findBySingleID(err error, packet nex.PacketInterface, callID uint32, id uint32) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonMatchMakingProtocol.server

	session, ok := common_globals.Sessions[id]
	if !ok {
		return nex.Errors.RendezVous.SessionVoid
	}

	bResult := true
	pGathering := nex.NewDataHolder()
	pGathering.SetTypeName("MatchmakeSession")
	pGathering.SetObjectData(session.GameMatchmakeSession)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteBool(bResult)
	rmcResponseStream.WriteDataHolder(pGathering)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodFindBySingleID
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	server.Send(responsePacket)

	return 0
}
