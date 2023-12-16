package nattraversal

import (
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func reportNATProperties(err error, packet nex.PacketInterface, callID uint32, natm uint32, natf uint32, rtt uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonProtocol.server

	// TODO - Remove cast to PRUDPClient?
	client := packet.Sender().(*nex.PRUDPClient)

	for _, station := range client.StationURLs {
		if station.IsLocal() {
			station.Fields.Set("natm", strconv.Itoa(int(natm)))
			station.Fields.Set("natf", strconv.Itoa(int(natf)))
		}

		station.Fields.Set("RVCID", strconv.Itoa(int(client.ConnectionID)))
		station.Fields.Set("PID", strconv.Itoa(int(client.PID().Value())))
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodReportNATProperties
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
