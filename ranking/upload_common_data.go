package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func uploadCommonData(err error, packet nex.PacketInterface, callID uint32, commonData []byte, uniqueID uint64) uint32 {
	if commonRankingProtocol.uploadCommonDataHandler == nil {
		common_globals.Logger.Warning("Ranking::UploadCommonData missing UploadCommonDataHandler!")
		return nex.Errors.Core.NotImplemented
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonRankingProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Ranking.InvalidArgument
	}

	err = commonRankingProtocol.uploadCommonDataHandler(client.PID(), uniqueID, commonData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nex.Errors.Ranking.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodUploadCommonData
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	server.Send(responsePacket)

	return 0
}
