package ranking

import (
	"strings"

	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_mario_kart_8 "github.com/PretendoNetwork/nex-protocols-go/ranking/mario-kart-8"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonRankingProtocol *CommonRankingProtocol

type CommonRankingProtocol struct {
	server             nex.ServerInterface
	DefaultProtocol    *ranking.Protocol
	MarioKart8Protocol *ranking_mario_kart_8.Protocol

	getCommonDataHandler                                     func(unique_id uint64) ([]byte, error)
	uploadCommonDataHandler                                  func(pid uint32, uniqueID uint64, commonData []byte) error
	insertRankingByPIDAndRankingScoreDataHandler             func(pid uint32, rankingScoreData *ranking_types.RankingScoreData, uniqueID uint64) error
	getRankingsAndCountByCategoryAndRankingOrderParamHandler func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) ([]*ranking_types.RankingRankData, uint32, error)
}

// GetCommonData sets the GetCommonData handler function
func (commonRankingProtocol *CommonRankingProtocol) GetCommonData(handler func(unique_id uint64) ([]byte, error)) {
	commonRankingProtocol.getCommonDataHandler = handler
}

// UploadCommonData sets the UploadCommonData handler function
func (commonRankingProtocol *CommonRankingProtocol) UploadCommonData(handler func(pid uint32, uniqueID uint64, commonData []byte) error) {
	commonRankingProtocol.uploadCommonDataHandler = handler
}

// InsertRankingByPIDAndRankingScoreData sets the InsertRankingByPIDAndRankingScoreData handler function
func (commonRankingProtocol *CommonRankingProtocol) InsertRankingByPIDAndRankingScoreData(handler func(pid uint32, rankingScoreData *ranking_types.RankingScoreData, uniqueID uint64) error) {
	commonRankingProtocol.insertRankingByPIDAndRankingScoreDataHandler = handler
}

// GetRankingsAndCountByCategoryAndRankingOrderParam sets the GetRankingsAndCountByCategoryAndRankingOrderParam handler function
func (commonRankingProtocol *CommonRankingProtocol) GetRankingsAndCountByCategoryAndRankingOrderParam(handler func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) ([]*ranking_types.RankingRankData, uint32, error)) {
	commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler = handler
}

func initDefault(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.DefaultProtocol = ranking.NewProtocol(c.server)
	c.DefaultProtocol.GetCachedTopXRanking = getCachedTopXRanking
	c.DefaultProtocol.GetCachedTopXRankings = getCachedTopXRankings
	c.DefaultProtocol.GetCommonData = getCommonData
	c.DefaultProtocol.GetRanking = getRanking
	c.DefaultProtocol.UploadCommonData = uploadCommonData
	c.DefaultProtocol.UploadScore = uploadScore
}

func initMarioKart8(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.MarioKart8Protocol = ranking_mario_kart_8.NewProtocol(c.server)
	c.MarioKart8Protocol.GetCachedTopXRanking = getCachedTopXRanking
	c.MarioKart8Protocol.GetCachedTopXRankings = getCachedTopXRankings
	c.MarioKart8Protocol.GetCommonData = getCommonData
	c.MarioKart8Protocol.GetRanking = getRanking
	c.MarioKart8Protocol.UploadCommonData = uploadCommonData
	c.MarioKart8Protocol.UploadScore = uploadScore
}

// NewCommonRankingProtocol returns a new CommonRankingProtocol
func NewCommonRankingProtocol(server nex.ServerInterface) *CommonRankingProtocol {
	commonRankingProtocol = &CommonRankingProtocol{server: server}

	patch := server.MatchMakingProtocolVersion().GameSpecificPatch

	if strings.EqualFold(patch, "AMKJ") {
		common_globals.Logger.Info("Using Mario Kart 8 Ranking protocol")
		initMarioKart8(commonRankingProtocol)
	} else {
		if patch != "" {
			common_globals.Logger.Infof("Ranking version patch %q not recognized", patch)
		}

		common_globals.Logger.Info("Using default Ranking protocol")
		initDefault(commonRankingProtocol)
	}

	return commonRankingProtocol
}
