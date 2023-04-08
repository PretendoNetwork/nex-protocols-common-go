package secureconnection

import (
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

func register(err error, client *nex.Client, callID uint32, stationUrls []*nex.StationURL) {
	server := commonSecureConnectionProtocol.server
	missingHandler := false
	if commonSecureConnectionProtocol.addConnectionHandler == nil {
		logger.Warning("Missing AddConnectionHandler!")
		missingHandler = true
	}
	if commonSecureConnectionProtocol.updateConnectionHandler == nil {
		logger.Warning("Missing UpdateConnectionHandler!")
		missingHandler = true
	}
	if commonSecureConnectionProtocol.doesConnectionExistHandler == nil {
		logger.Warning("Missing DoesConnectionExistHandler!")
		missingHandler = true
	}
	if missingHandler {
		return
	}
	localStation := stationUrls[0]
	localStationURL := localStation.EncodeToString()
	pidConnectionID := uint32(server.ConnectionIDCounter().Increment())
	client.SetConnectionID(pidConnectionID)
	client.SetLocalStationURL(localStationURL)

	address := client.Address().IP.String()
	port := strconv.Itoa(client.Address().Port)
	natf := "0"
	natm := "0"
	type_ := "3"

	localStation.SetAddress(address)
	localStation.SetPort(port)
	localStation.SetNatf(natf)
	localStation.SetNatm(natm)
	localStation.SetType(type_)

	urlPublic := localStation.EncodeToString()

	if !commonSecureConnectionProtocol.doesConnectionExistHandler(pidConnectionID) {
		commonSecureConnectionProtocol.addConnectionHandler(pidConnectionID, []string{localStationURL, urlPublic}, address, port)
	} else {
		commonSecureConnectionProtocol.updateConnectionHandler(pidConnectionID, []string{localStationURL, urlPublic}, address, port)
	}

	retval := nex.NewResultSuccess(nex.Errors.Core.Unknown)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteResult(retval) // Success
	rmcResponseStream.WriteUInt32LE(pidConnectionID)
	rmcResponseStream.WriteString(urlPublic)

	rmcResponseBody := rmcResponseStream.Bytes()

	// Build response packet
	rmcResponse := nex.NewRMCResponse(secure_connection.ProtocolID, callID)
	rmcResponse.SetSuccess(secure_connection.MethodRegister, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)
}
