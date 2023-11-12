package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getCommonData(err error, packet nex.PacketInterface, callID uint32, uniqueID uint64) uint32 {
	if commonRankingProtocol.getCommonDataHandler == nil {
		common_globals.Logger.Warning("Ranking::GetCommonData missing GetCommonDataHandler!")
		return nex.Errors.Core.NotImplemented
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonRankingProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Ranking.InvalidArgument
	}

	commonData, err := commonRankingProtocol.getCommonDataHandler(uniqueID)
	if err != nil {
		return nex.Errors.Ranking.NotFound
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBuffer(commonData)
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCommonData
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
