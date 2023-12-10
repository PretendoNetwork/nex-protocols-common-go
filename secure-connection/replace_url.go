package secureconnection

import (
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func replaceURL(err error, packet nex.PacketInterface, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - Remove cast to PRUDPClient once websockets are implemented
	client := packet.Sender().(*nex.PRUDPClient)

	stations := client.StationURLs
	for i := 0; i < len(stations); i++ {
		currentStation := stations[i]

		currentStationAddress, currentStationAddressOk := currentStation.Fields.Get("address")
		currentStationPort, currentStationPortOk := currentStation.Fields.Get("port")
		oldStationAddress, oldStationAddressOk := oldStation.Fields.Get("address")
		oldStationPort, oldStationPortOk := oldStation.Fields.Get("port")

		if currentStationAddressOk && currentStationPortOk && oldStationAddressOk && oldStationPortOk {
			if currentStationAddress == oldStationAddress && currentStationPort == oldStationPort {
				// * This fixes Minecraft, but is obviously incorrect
				// TODO - What are we really meant to do here?
				newStation.Fields.Set("PID", strconv.Itoa(int(client.PID().Value())))
				stations[i] = newStation
			}
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodReplaceURL
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
