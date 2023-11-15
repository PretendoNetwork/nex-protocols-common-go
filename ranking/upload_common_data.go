package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func uploadCommonData(err error, packet nex.PacketInterface, callID uint32, commonData []byte, uniqueID uint64) (*nex.RMCMessage, uint32) {
	if commonRankingProtocol.uploadCommonDataHandler == nil {
		common_globals.Logger.Warning("Ranking::UploadCommonData missing UploadCommonDataHandler!")
		return nil, nex.Errors.Core.NotImplemented
	}

	client := packet.Sender().(*nex.PRUDPClient)

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	err = commonRankingProtocol.uploadCommonDataHandler(client.PID().LegacyValue(), uniqueID, commonData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Ranking.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodUploadCommonData
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
