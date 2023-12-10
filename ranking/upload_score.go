package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func uploadScore(err error, packet nex.PacketInterface, callID uint32, scoreData *ranking_types.RankingScoreData, uniqueID uint64) (*nex.RMCMessage, uint32) {
	if commonRankingProtocol.InsertRankingByPIDAndRankingScoreData == nil {
		common_globals.Logger.Warning("Ranking::UploadScore missing InsertRankingByPIDAndRankingScoreData!")
		return nil, nex.Errors.Core.NotImplemented
	}

	client := packet.Sender()

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	err = commonRankingProtocol.InsertRankingByPIDAndRankingScoreData(client.PID().LegacyValue(), scoreData, uniqueID)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Ranking.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodUploadScore
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
