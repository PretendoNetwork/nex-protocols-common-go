package utility

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	utility "github.com/PretendoNetwork/nex-protocols-go/utility"
)

func acquireNexUniqueID(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	if commonProtocol.GenerateNEXUniqueID == nil {
		common_globals.Logger.Warning("Utility::AcquireNexUniqueID missing GenerateNEXUniqueID!")
		return nil, nex.ResultCodesCore.NotImplemented
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	pNexUniqueID := types.NewPrimitiveU64(commonProtocol.GenerateNEXUniqueID())

	rmcResponseStream := nex.NewByteStreamOut(server)

	pNexUniqueID.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = utility.ProtocolID
	rmcResponse.MethodID = utility.MethodAcquireNexUniqueID
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
