package secureconnection

import (
	"net"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func register(err error, packet nex.PacketInterface, callID uint32, stationUrls []*nex.StationURL) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonSecureConnectionProtocol.server

	// TODO - Remove cast to PRUDPClient once websockets are implemented
	client := packet.Sender().(*nex.PRUDPClient)

	client.ConnectionID = server.ConnectionIDCounter().Next()

	localStation := stationUrls[0]

	// * A NEX client can set the public station URL by setting two URLs on the array
	// * Check each URL for a public station
	var publicStation *nex.StationURL
	for _, stationURL := range stationUrls {
		if transportType, ok := stationURL.Fields.Get("type"); ok {
			if transportType == "3" {
				publicStation = stationURL
				break
			}
		}
	}

	if publicStation == nil {
		publicStation = localStation.Copy()

		publicStation.Fields.Set("address", client.Address().(*net.UDPAddr).IP.String())
		publicStation.Fields.Set("port", strconv.Itoa(client.Address().(*net.UDPAddr).Port))
		publicStation.Fields.Set("natf", "0")
		publicStation.Fields.Set("natm", "0")
		publicStation.Fields.Set("type", "3")
	}

	localStation.Fields.Set("PID", strconv.Itoa(int(client.PID().Value())))
	publicStation.Fields.Set("PID", strconv.Itoa(int(client.PID().Value())))

	localStation.Fields.Set("RVCID", strconv.Itoa(int(client.ConnectionID)))
	publicStation.Fields.Set("RVCID", strconv.Itoa(int(client.ConnectionID)))

	localStation.SetLocal()
	publicStation.SetPublic()

	client.StationURLs = append(client.StationURLs, localStation)
	client.StationURLs = append(client.StationURLs, publicStation)

	retval := nex.NewResultSuccess(nex.Errors.Core.Unknown)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteResult(retval)
	rmcResponseStream.WriteUInt32LE(client.ConnectionID)
	rmcResponseStream.WriteString(publicStation.EncodeToString())

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = secure_connection.ProtocolID
	rmcResponse.MethodID = secure_connection.MethodRegister
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
