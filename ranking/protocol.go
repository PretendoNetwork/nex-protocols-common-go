package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server                                            nex.ServerInterface
	protocol                                          ranking.Interface
	GetCommonData                                     func(unique_id uint64) ([]byte, error)
	UploadCommonData                                  func(pid uint32, uniqueID uint64, commonData []byte) error
	InsertRankingByPIDAndRankingScoreData             func(pid uint32, rankingScoreData *ranking_types.RankingScoreData, uniqueID uint64) error
	GetRankingsAndCountByCategoryAndRankingOrderParam func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) ([]*ranking_types.RankingRankData, uint32, error)
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
