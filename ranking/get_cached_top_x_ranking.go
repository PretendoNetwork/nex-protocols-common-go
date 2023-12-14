package ranking

import (
	"time"

	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getCachedTopXRanking(err error, packet nex.PacketInterface, callID uint32, category uint32, orderParam *ranking_types.RankingOrderParam) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetCachedTopXRanking missing GetRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.Errors.Core.NotImplemented
	}

	server := commonProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	rankDataList, totalCount, err := commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam(category, orderParam)
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

	pResult := ranking_types.NewRankingCachedResult()

	pResult.SetParentType(rankingResult)
	pResult.CreatedTime = nex.NewDateTime(0).Now()
	// * The real server sends the "CreatedTime" + 5 minutes.
	// * It doesn't change, even on subsequent requests, until after the
	// * ExpiredTime has passed (seemingly what the "cached" means).
	// * Whether we need to replicate this idk, but in case, here's a note.
	pResult.ExpiredTime = nex.NewDateTime(0).FromTimestamp(time.Now().UTC().Add(time.Minute * time.Duration(5)))
	// * This is the length Ultimate NES Remix uses
	// TODO - Does this matter? and are other games different?
	pResult.MaxLength = 10

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteStructure(pResult)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCachedTopXRanking
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
