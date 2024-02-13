package nattraversal

import (
	"strconv"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func (commonProtocol *CommonProtocol) reportNATProperties(err error, packet nex.PacketInterface, callID uint32, natmapping *types.PrimitiveU32, natfiltering *types.PrimitiveU32, rtt *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	for _, station := range connection.StationURLs.Slice() {
		if station.IsLocal() {
			station.Fields["natm"] = strconv.Itoa(int(natmapping.Value))
			station.Fields["natf"] = strconv.Itoa(int(natfiltering.Value))
		}

		station.Fields["RVCID"] = strconv.Itoa(int(connection.ID))
		station.Fields["PID"] = strconv.Itoa(int(connection.PID().Value()))
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodReportNATProperties
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterReportNATProperties != nil {
		go commonProtocol.OnAfterReportNATProperties(packet, natmapping, natfiltering, rtt)
	}

	return rmcResponse, nil
}
