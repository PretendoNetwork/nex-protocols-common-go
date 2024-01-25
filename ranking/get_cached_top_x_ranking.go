package ranking

import (
	"time"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

func getCachedTopXRanking(err error, packet nex.PacketInterface, callID uint32, category *types.PrimitiveU32, orderParam *ranking_types.RankingOrderParam) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam == nil {
		common_globals.Logger.Warning("Ranking::GetCachedTopXRanking missing GetRankingsAndCountByCategoryAndRankingOrderParam!")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Ranking.InvalidArgument
	}

	rankDataList, totalCount, err := commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam(category, orderParam)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.ResultCodes.Ranking.Unknown
	}

	if totalCount == 0 || rankDataList.Length() == 0 {
		return nil, nex.ResultCodes.Ranking.NotFound
	}

	pResult := ranking_types.NewRankingCachedResult()

	pResult.RankingResult.RankDataList = rankDataList
	pResult.RankingResult.TotalCount = types.NewPrimitiveU32(totalCount)
	pResult.RankingResult.SinceTime = types.NewDateTime(0x1F40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

	pResult.CreatedTime = types.NewDateTime(0).Now()
	// * The real server sends the "CreatedTime" + 5 minutes.
	// * It doesn't change, even on subsequent requests, until after the
	// * ExpiredTime has passed (seemingly what the "cached" means).
	// * Whether we need to replicate this idk, but in case, here's a note.
	pResult.ExpiredTime = types.NewDateTime(0).FromTimestamp(time.Now().UTC().Add(time.Minute * time.Duration(5)))
	// * This is the length Ultimate NES Remix uses
	// TODO - Does this matter? and are other games different?
	pResult.MaxLength = types.NewPrimitiveU8(10)

	rmcResponseStream := nex.NewByteStreamOut(server)

	pResult.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCachedTopXRanking
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
