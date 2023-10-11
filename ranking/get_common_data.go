package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
)

func getCommonData(err error, client *nex.Client, callID uint32, uniqueID uint64) uint32 {
	if commonRankingProtocol.getCommonDataHandler == nil {
		logger.Warning("Ranking::GetCommonData missing GetCommonDataHandler!")
		return nex.Errors.Core.NotImplemented
	}

	server := client.Server()

	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Ranking.InvalidArgument
	}

	commonData, err := commonRankingProtocol.getCommonDataHandler(uniqueID)

	if err != nil {
		return nex.Errors.Ranking.NotFound
	}
		
	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBuffer(commonData)
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	rmcResponse.SetSuccess(ranking.MethodGetCommonData, rmcResponseBody)

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
