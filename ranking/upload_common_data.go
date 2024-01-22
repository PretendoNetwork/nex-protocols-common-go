package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
)

func uploadCommonData(err error, packet nex.PacketInterface, callID uint32, commonData *types.Buffer, uniqueID *types.PrimitiveU64) (*nex.RMCMessage, uint32) {
	if commonProtocol.UploadCommonData == nil {
		common_globals.Logger.Warning("Ranking::UploadCommonData missing UploadCommonData!")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Ranking.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	err = commonProtocol.UploadCommonData(connection.PID(), uniqueID, commonData)
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
