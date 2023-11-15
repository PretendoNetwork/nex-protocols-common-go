package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func reportNATProperties(err error, packet nex.PacketInterface, callID uint32, natm uint32, natf uint32, rtt uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)

	for _, station := range client.StationURLs {
		if station.IsLocal() {
			station.SetNatm(natm)
			station.SetNatf(natf)
		}

		station.SetRVCID(client.ConnectionID)
		station.SetPID(client.PID())
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodReportNATProperties
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
