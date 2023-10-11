package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
)

func uploadCommonData(err error, client *nex.Client, callID uint32, commonData []byte, uniqueID uint64) uint32 {
	if commonRankingProtocol.uploadCommonDataHandler == nil {
		logger.Warning("Ranking::UploadCommonData missing UploadCommonDataHandler!")
		return nex.Errors.Core.NotImplemented
	}
	rmcResponse := nex.NewRMCResponse(ranking.ProtocolID, callID)
	server := client.Server()

	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Ranking.InvalidArgument
	}

	insertErr := commonRankingProtocol.uploadCommonDataHandler(client.PID(), uniqueID, commonData)
	if insertErr != nil {
		logger.Critical(insertErr.Error())
		return nex.Errors.Ranking.Unknown
	}
	
	rmcResponse.SetSuccess(ranking.MethodUploadCommonData, nil)

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
