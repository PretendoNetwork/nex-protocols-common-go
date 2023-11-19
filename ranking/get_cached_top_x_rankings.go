package ranking

import (
	"time"

	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getCachedTopXRankings(err error, packet nex.PacketInterface, callID uint32, categories []uint32, orderParams []*ranking_types.RankingOrderParam) (*nex.RMCMessage, uint32) {
	if commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler == nil {
		common_globals.Logger.Warning("Ranking::GetCachedTopXRankings missing GetRankingsAndCountByCategoryAndRankingOrderParamHandler!")
		return nil, nex.Errors.Core.NotImplemented
	}

	server := commonRankingProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	var pResult []*ranking_types.RankingCachedResult
	for i := 0; i < len(categories); i++ {
		rankDataList, totalCount, err := commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler(categories[i], orderParams[i])
		if err != nil {
			common_globals.Logger.Critical(err.Error())
			return nil, nex.Errors.Ranking.Unknown
		}

		if totalCount == 0 || len(rankDataList) == 0 {
			return nil, nex.Errors.Ranking.NotFound
		}

		rankingResult := ranking_types.NewRankingResult()

		rankingResult.RankDataList = rankDataList
		rankingResult.TotalCount = totalCount
		rankingResult.SinceTime = nex.NewDateTime(0x1F40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

		result := ranking_types.NewRankingCachedResult()

		result.SetParentType(rankingResult)
		result.CreatedTime = nex.NewDateTime(0).Now()
		// * The real server sends the "CreatedTime" + 5 minutes.
		// * It doesn't change, even on subsequent requests, until after the
		// * ExpiredTime has passed (seemingly what the "cached" means).
		// * Whether we need to replicate this idk, but in case, here's a note.
		result.ExpiredTime = nex.NewDateTime(0).FromTimestamp(time.Now().UTC().Add(time.Minute * time.Duration(5)))
		// * This is the length Ultimate NES Remix uses
		// TODO - Does this matter? and are other games different?
		result.MaxLength = 10

		pResult = append(pResult, result)
	}

	rmcResponseStream := nex.NewStreamOut(server)
	nex.StreamWriteListStructure(rmcResponseStream, pResult)
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCachedTopXRankings
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
