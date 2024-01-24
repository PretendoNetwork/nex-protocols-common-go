package secureconnection

import (
	"strconv"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func replaceURL(err error, packet nex.PacketInterface, callID uint32, target *types.StationURL, url *types.StationURL) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	connection.StationURLs.Each(func(i int, station *types.StationURL) bool {
		currentStationAddress, currentStationAddressOk := station.Fields["address"]
		currentStationPort, currentStationPortOk := station.Fields["port"]
		oldStationAddress, oldStationAddressOk := target.Fields["address"]
		oldStationPort, oldStationPortOk := target.Fields["port"]

		if currentStationAddressOk && currentStationPortOk && oldStationAddressOk && oldStationPortOk {
			if currentStationAddress == oldStationAddress && currentStationPort == oldStationPort {
				// * This fixes Minecraft, but is obviously incorrect
				// TODO - What are we really meant to do here?
				newStation := url.Copy().(*types.StationURL)

				newStation.Fields["PID"] = strconv.Itoa(int(connection.PID().Value()))

				connection.StationURLs.SetIndex(i, newStation)
			}
		}

		return false
	})

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodReplaceURL
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
