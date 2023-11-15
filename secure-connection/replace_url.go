package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func replaceURL(err error, packet nex.PacketInterface, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)

	stations := client.StationURLs
	for i := 0; i < len(stations); i++ {
		currentStation := stations[i]
		if currentStation.Address() == oldStation.Address() && currentStation.Port() == oldStation.Port() {
			// * This fixes Minecraft, but is obviously incorrect
			// TODO - What are we really meant to do here?
			newStation.SetPID(client.PID())
			stations[i] = newStation
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodReplaceURL
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
