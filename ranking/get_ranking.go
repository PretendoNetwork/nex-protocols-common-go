package ranking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/v2/ranking"
	"github.com/PretendoNetwork/nex-protocols-go/v2/ranking/constants"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/types"
)

func (commonProtocol *CommonProtocol) getRanking(err error, packet nex.PacketInterface, callID uint32, rankingMode types.UInt8, category types.UInt32, orderParam ranking_types.RankingOrderParam, uniqueID types.UInt64, principalID types.PID) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Ranking.InvalidArgument, "change_error")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()
	callerPID := principalID
	// * 0 = "own ranking"
	if callerPID == 0 {
		callerPID = connection.PID()
	}

	var rankDataList types.List[ranking_types.RankingRankData]
	var totalCount uint32

	switch constants.RankingMode(rankingMode) {
	case constants.RankingModeRange:
		if commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam == nil {
			common_globals.Logger.Warning("Ranking::GetRanking missing GetRankingsAndCountByCategoryAndRankingOrderParam!")
			return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		rankDataList, totalCount, err = commonProtocol.GetRankingsAndCountByCategoryAndRankingOrderParam(category, orderParam)
	case constants.RankingModeNear:
		if commonProtocol.GetNearbyRankingsAndCountByCategoryAndRankingOrderParam == nil {
			common_globals.Logger.Warning("Ranking::GetRanking missing GetNearbyRankingsAndCountByCategoryAndRankingOrderParam!")
			return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		rankDataList, totalCount, err = commonProtocol.GetNearbyRankingsAndCountByCategoryAndRankingOrderParam(callerPID, category, orderParam)
	case constants.RankingModeFriendRange:
		if commonProtocol.GetFriendsRankingsAndCountByCategoryAndRankingOrderParam == nil {
			common_globals.Logger.Warning("Ranking::GetRanking missing GetFriendsRankingsAndCountByCategoryAndRankingOrderParam!")
			return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		rankDataList, totalCount, err = commonProtocol.GetFriendsRankingsAndCountByCategoryAndRankingOrderParam(callerPID, category, orderParam)
	case constants.RankingModeFriendNear:
		if commonProtocol.GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam == nil {
			common_globals.Logger.Warning("Ranking::GetRanking missing GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam!")
			return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		rankDataList, totalCount, err = commonProtocol.GetNearbyFriendsRankingsAndCountByCategoryAndRankingOrderParam(callerPID, category, orderParam)
	case constants.RankingModeUser:
		if commonProtocol.GetOwnRankingByCategoryAndRankingOrderParam == nil {
			common_globals.Logger.Warning("Ranking::GetRanking missing GetOwnRankingByCategoryAndRankingOrderParam!")
			return nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		rankDataList, totalCount, err = commonProtocol.GetOwnRankingByCategoryAndRankingOrderParam(callerPID, category, orderParam)
	default:
		common_globals.Logger.Errorf("Unknown RankingMode %v!", rankingMode)
		return nil, nex.NewError(nex.ResultCodes.Ranking.InvalidArgument, "Unknown RankingMode")
	}

	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Ranking.Unknown, "change_error")
	}

	if totalCount == 0 || len(rankDataList) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Ranking.NotFound, "change_error")
	}

	pResult := ranking_types.NewRankingResult()

	pResult.RankDataList = rankDataList
	pResult.TotalCount = types.NewUInt32(totalCount)
	pResult.SinceTime = types.NewDateTime(0x1F40420000) // * 2000-01-01T00:00:00.000Z, this is what the real server sends back

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pResult.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetRanking
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetRanking != nil {
		go commonProtocol.OnAfterGetRanking(packet, rankingMode, category, orderParam, uniqueID, principalID)
	}

	return rmcResponse, nil
}
