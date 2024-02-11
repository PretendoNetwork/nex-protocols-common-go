package nattraversal

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func reportNATTraversalResult(err error, packet nex.PacketInterface, callID uint32, cid *types.PrimitiveU32, result *types.PrimitiveBool, rtt *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodReportNATTraversalResult
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
