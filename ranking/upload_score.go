package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

func uploadScore(err error, packet nex.PacketInterface, callID uint32, scoreData *ranking_types.RankingScoreData, uniqueID *types.PrimitiveU64) (*nex.RMCMessage, uint32) {
	if commonProtocol.InsertRankingByPIDAndRankingScoreData == nil {
		common_globals.Logger.Warning("Ranking::UploadScore missing InsertRankingByPIDAndRankingScoreData!")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Ranking.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	err = commonProtocol.InsertRankingByPIDAndRankingScoreData(connection.PID(), scoreData, uniqueID)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.ResultCodes.Ranking.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodUploadScore
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
