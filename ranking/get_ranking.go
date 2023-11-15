package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getRanking(err error, packet nex.PacketInterface, callID uint32, rankingMode uint8, category uint32, orderParam *ranking_types.RankingOrderParam, uniqueID uint64, principalID *nex.PID) (*nex.RMCMessage, uint32) {
	if commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler == nil {
		common_globals.Logger.Warning("Ranking::GetRanking missing GetRankingsAndCountByCategoryAndRankingOrderParamHandler!")
		return nil, nex.Errors.Core.NotImplemented
	}

	server := commonRankingProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	rankDataList, totalCount, err := commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler(category, orderParam)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Ranking.Unknown
	}

	if totalCount == 0 || len(rankDataList) == 0 {
		return nil, nex.Errors.Ranking.NotFound
	}

	pResult := ranking_types.NewRankingResult()

	pResult.RankDataList = rankDataList
	pResult.TotalCount = totalCount
	pResult.SinceTime = nex.NewDateTime(0x1f40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(pResult)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetRanking
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
