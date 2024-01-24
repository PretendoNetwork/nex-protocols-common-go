package nattraversal

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func reportNATTraversalResult(err error, packet nex.PacketInterface, callID uint32, cid *types.PrimitiveU32, result *types.PrimitiveBool, rtt *types.PrimitiveU32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodReportNATTraversalResult
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
