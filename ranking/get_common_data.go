package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func getCommonData(err error, packet nex.PacketInterface, callID uint32, uniqueID uint64) (*nex.RMCMessage, uint32) {
	if commonRankingProtocol.getCommonDataHandler == nil {
		common_globals.Logger.Warning("Ranking::GetCommonData missing GetCommonDataHandler!")
		return nil, nex.Errors.Core.NotImplemented
	}

	server := commonRankingProtocol.server

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	commonData, err := commonRankingProtocol.getCommonDataHandler(uniqueID)
	if err != nil {
		return nil, nex.Errors.Ranking.NotFound
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBuffer(commonData)
	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCommonData
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
