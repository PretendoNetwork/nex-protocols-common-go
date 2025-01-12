package nattraversal

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/v2/nat-traversal"
)

func (commonProtocol *CommonProtocol) reportNATProperties(err error, packet nex.PacketInterface, callID uint32, natmapping types.UInt32, natfiltering types.UInt32, rtt types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	for _, station := range connection.StationURLs {
		if !station.IsPublic() {
			station.SetNATMapping(constants.NATMappingProperties(natmapping))
			station.SetNATFiltering(constants.NATFilteringProperties(natfiltering))
		}

		station.SetRVConnectionID(connection.ID)
		station.SetPrincipalID(connection.PID())
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
