package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server                                            nex.ServerInterface
	protocol                                          ranking.Interface
	GetCommonData                                     func(uniqueID *types.PrimitiveU64) (*types.Buffer, error)
	UploadCommonData                                  func(pid *types.PID, uniqueID *types.PrimitiveU64, commonData *types.Buffer) error
	InsertRankingByPIDAndRankingScoreData             func(pid *types.PID, rankingScoreData *ranking_types.RankingScoreData, uniqueID *types.PrimitiveU64) error
	GetRankingsAndCountByCategoryAndRankingOrderParam func(category *types.PrimitiveU32, rankingOrderParam *ranking_types.RankingOrderParam) (*types.List[*ranking_types.RankingRankData], uint32, error)
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol ranking.Interface) *CommonProtocol {
	protocol.SetHandlerGetCachedTopXRanking(getCachedTopXRanking)
	protocol.SetHandlerGetCachedTopXRankings(getCachedTopXRankings)
	protocol.SetHandlerGetCommonData(getCommonData)
	protocol.SetHandlerGetRanking(getRanking)
	protocol.SetHandlerUploadCommonData(uploadCommonData)
	protocol.SetHandlerUploadScore(uploadScore)

	commonProtocol = &CommonProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonProtocol
}
