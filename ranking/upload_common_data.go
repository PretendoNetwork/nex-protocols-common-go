package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func uploadCommonData(err error, packet nex.PacketInterface, callID uint32, commonData []byte, uniqueID uint64) (*nex.RMCMessage, uint32) {
	if commonProtocol.UploadCommonData == nil {
		common_globals.Logger.Warning("Ranking::UploadCommonData missing UploadCommonData!")
		return nil, nex.Errors.Core.NotImplemented
	}

	server := commonProtocol.server
	client := packet.Sender()

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	err = commonProtocol.UploadCommonData(client.PID().LegacyValue(), uniqueID, commonData)
	if err != nil {
		common_globals.Logger.Critical(err.Error())
		return nil, nex.Errors.Ranking.Unknown
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodUploadCommonData
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
