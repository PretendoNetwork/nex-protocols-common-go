package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
	ranking_types "github.com/PretendoNetwork/nex-protocols-go/ranking/types"
)

func uploadScore(err error, client *nex.Client, callID uint32, scoreData *ranking_types.RankingScoreData, uniqueID uint64) uint32 {
	if commonRankingProtocol.insertRankingByPIDAndRankingScoreDataHandler == nil {
		logger.Warning("Ranking::UploadScore missing InsertRankingByPIDAndRankingScoreDataHandler!")
		return nex.Errors.Core.NotImplemented
	}
	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	server := client.Server()

	if err != nil {
		logger.Error(err.Error())
		rmcResponse.SetError(nex.Errors.Ranking.Unknown)
	}

	insertErr := commonRankingProtocol.insertRankingByPIDAndRankingScoreDataHandler(client.PID(), scoreData, uniqueID)
	if insertErr != nil {
		logger.Error(insertErr.Error())
		rmcResponse.SetError(nex.Errors.Ranking.Unknown)
	} else {
		rmcResponse.SetSuccess(ranking.MethodUploadScore, nil)
	}

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
