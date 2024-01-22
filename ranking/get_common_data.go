package ranking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ranking "github.com/PretendoNetwork/nex-protocols-go/ranking"
)

func getCommonData(err error, packet nex.PacketInterface, callID uint32, uniqueID *types.PrimitiveU64) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetCommonData == nil {
		common_globals.Logger.Warning("Ranking::GetCommonData missing GetCommonData!")
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

	commonData, err := commonProtocol.GetCommonData(uniqueID)
	if err != nil {
		return nil, nex.Errors.Ranking.NotFound
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	commonData.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = ranking.ProtocolID
	rmcResponse.MethodID = ranking.MethodGetCommonData
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
