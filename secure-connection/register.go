package secureconnection

import (
	"net"

	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func register(err error, packet nex.PacketInterface, callID uint32, stationUrls []*nex.StationURL) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonSecureConnectionProtocol.server
	client := packet.Sender().(*nex.PRUDPClient)

	client.ConnectionID = server.ConnectionIDCounter().Next()

	localStation := stationUrls[0]

	// * A NEX client can set the public station URL by setting two URLs on the array
	// * Check each URL for a public station
	var publicStation *nex.StationURL
	for _, stationURL := range stationUrls {
		if stationURL.Type() == 3 {
			publicStation = stationURL
			break
		}
	}

	if publicStation == nil {
		publicStation = localStation.Copy()

		publicStation.SetAddress(client.Address().(*net.UDPAddr).IP.String())
		publicStation.SetPort(uint32(client.Address().(*net.UDPAddr).Port))
		publicStation.SetNatf(0)
		publicStation.SetNatm(0)
		publicStation.SetType(3)
	}

	localStation.SetPID(client.PID())
	publicStation.SetPID(client.PID())

	localStation.SetRVCID(client.ConnectionID)
	publicStation.SetRVCID(client.ConnectionID)

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

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	server.Send(responsePacket)

	return 0
}
