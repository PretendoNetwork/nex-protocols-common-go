package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

func getCachedTopXRankings(err error, client *nex.Client, callID uint32, categories []uint32, orderParams []*ranking_types.RankingOrderParam) uint32 {
	if commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler == nil {
		logger.Warning("Ranking::GetCachedTopXRankings missing GetRankingsAndCountByCategoryAndRankingOrderParamHandler!")
		return nex.Errors.Core.NotImplemented
	}
	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	server := client.Server()

	if err != nil {
		logger.Error(err.Error())
		rmcResponse.SetError(nex.Errors.Ranking.Unknown)
	}
	
	var pResult []*ranking_types.RankingCachedResult
	for i := 0; i < len(categories); i++ {
		err, rankDataList, totalCount := commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler(categories[i], orderParams[i])
		if err != nil {
			logger.Error(err.Error())
			rmcResponse.SetError(nex.Errors.Ranking.Unknown)
		}

		if totalCount == 0 || len(rankDataList) == 0 {
			rmcResponse.SetError(nex.Errors.Ranking.NotFound)
		}

		if err == nil && totalCount != 0 {
			rankingResult := ranking_types.NewRankingResult()

			rankingResult.RankDataList = rankDataList
			rankingResult.TotalCount = totalCount
			rankingResult.SinceTime = nex.NewDateTime(0x1f40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back
			
			result := ranking_types.NewRankingCachedResult()
			result.CreatedTime = nex.NewDateTime(0x1f40420000) //TODO: does this matter?
			result.ExpiredTime = nex.NewDateTime(0x1f40420000) //TODO: does this matter?
			result.MaxLength = 0
			result.SetParentType(rankingResult)
			pResult = append(pResult, result)
		}

		rmcResponseStream := nex.NewStreamOut(server)
		rmcResponseStream.WriteListStructure(pResult)
		rmcResponseBody := rmcResponseStream.Bytes()
		rmcResponse.SetSuccess(ranking.MethodGetCachedTopXRankings, rmcResponseBody)
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
