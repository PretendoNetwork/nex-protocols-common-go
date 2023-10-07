package ranking

import (
	"strings"

	"github.com/PretendoNetwork/nex-go"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_mario_kart_8 "github.com/PretendoNetwork/nex-protocols-go/ranking/mario-kart-8"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
	"github.com/PretendoNetwork/plogger-go"
)

var commonRankingProtocol *CommonRankingProtocol
var logger = plogger.NewLogger()

type CommonRankingProtocol struct {
	server             *nex.Server
	DefaultProtocol    *ranking.Protocol
	MarioKart8Protocol *ranking_mario_kart_8.Protocol

	getCommonDataHandler                                     func(unique_id uint64) (error, []byte)
	uploadCommonDataHandler                                  func(pid uint32, uniqueID uint64, commonData []byte) error
	insertRankingByPIDAndRankingScoreDataHandler             func(pid uint32, rankingScoreData *ranking_types.RankingScoreData, uniqueID uint64) error
	getRankingsAndCountByCategoryAndRankingOrderParamHandler func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) (error, []*ranking_types.RankingRankData, uint32)
}

// GetCommonData sets the GetCommonData handler function
func (commonRankingProtocol *CommonRankingProtocol) GetCommonData(handler func(unique_id uint64) (error, []byte)) {
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
func (commonRankingProtocol *CommonRankingProtocol) GetRankingsAndCountByCategoryAndRankingOrderParam(handler func(category uint32, rankingOrderParam *ranking_types.RankingOrderParam) (error, []*ranking_types.RankingRankData, uint32)) {
	commonRankingProtocol.getRankingsAndCountByCategoryAndRankingOrderParamHandler = handler
}

func initDefault(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.DefaultProtocol = ranking.NewProtocol(c.server)
	c.DefaultProtocol.GetCachedTopXRanking(getCachedTopXRanking)
	c.DefaultProtocol.GetCachedTopXRankings(getCachedTopXRankings)
	c.DefaultProtocol.GetCommonData(getCommonData)
	c.DefaultProtocol.GetRanking(getRanking)
	c.DefaultProtocol.UploadCommonData(uploadCommonData)
	c.DefaultProtocol.UploadScore(uploadScore)
}

func initMarioKart8(c *CommonRankingProtocol) {
	// TODO - Organize by method ID
	c.MarioKart8Protocol = ranking_mario_kart_8.NewProtocol(c.server)
	c.MarioKart8Protocol.GetCachedTopXRanking(getCachedTopXRanking)
	c.MarioKart8Protocol.GetCachedTopXRankings(getCachedTopXRankings)
	c.MarioKart8Protocol.GetCommonData(getCommonData)
	c.MarioKart8Protocol.GetRanking(getRanking)
	c.MarioKart8Protocol.UploadCommonData(uploadCommonData)
	c.MarioKart8Protocol.UploadScore(uploadScore)
}

// NewCommonRankingProtocol returns a new CommonRankingProtocol
func NewCommonRankingProtocol(server *nex.Server) *CommonRankingProtocol {
	commonRankingProtocol = &CommonRankingProtocol{server: server}

	patch := server.MatchMakingProtocolVersion().GameSpecificPatch

	if strings.EqualFold(patch, "AMKJ") {
		logger.Info("Using Mario Kart 8 Ranking protocol")
		initMarioKart8(commonRankingProtocol)
	} else {
		if patch != "" {
			logger.Infof("Ranking version patch %q not recognized", patch)
		}

		logger.Info("Using default Ranking protocol")
		initDefault(commonRankingProtocol)
	}

	return commonRankingProtocol
}
