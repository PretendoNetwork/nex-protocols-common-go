package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

func getRanking(err error, client *nex.Client, callID uint32, rankingMode uint8, category uint32, orderParam *ranking_types.RankingOrderParam, uniqueID uint64, principalID uint32) uint32 {
	if commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler == nil {
		logger.Warning("Ranking::GetRanking missing GetRankingsAndCountByCategoryAndRankingOrderParamHandler!")
		return nex.Errors.Core.NotImplemented
	}
	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	server := client.Server()

	if err != nil {
		logger.Error(err.Error())
		rmcResponse.SetError(nex.Errors.Ranking.Unknown)
	}

	err, rankDataList, totalCount := commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler(category, orderParam)
	if err != nil {
		logger.Error(err.Error())
		rmcResponse.SetError(nex.Errors.Ranking.Unknown)
	}

	if totalCount == 0 || len(rankDataList) == 0 {
		rmcResponse.SetError(nex.Errors.Ranking.NotFound)
	}

	if err == nil && totalCount != 0 {
		pResult := ranking_types.NewRankingResult()

		pResult.RankDataList = rankDataList
		pResult.TotalCount = totalCount
		pResult.SinceTime = nex.NewDateTime(0x1f40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

		rmcResponseStream := nex.NewStreamOut(server)

		rmcResponseStream.WriteStructure(pResult)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse.SetSuccess(ranking.MethodGetRanking, rmcResponseBody)
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

	return 0
}
