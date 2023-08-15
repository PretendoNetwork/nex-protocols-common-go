package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

func register(err error, client *nex.Client, callID uint32, stationUrls []*nex.StationURL) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonSecureConnectionProtocol.server

	nextConnectionID := uint32(server.ConnectionIDCounter().Increment())
	client.SetConnectionID(nextConnectionID)

	localStation := stationUrls[0]
	publicStation := localStation.Copy()

	publicStation.SetAddress(client.Address().IP.String())
	publicStation.SetPort(uint32(client.Address().Port))
	publicStation.SetNatf(0)
	publicStation.SetNatm(0)
	publicStation.SetType(3)
	publicStation.SetPID(client.PID())

	localStation.SetLocal()
	publicStation.SetPublic()

	client.AddStationURL(localStation)
	client.AddStationURL(publicStation)

	retval := nex.NewResultSuccess(nex.Errors.Core.Unknown)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteResult(retval) // Success
	rmcResponseStream.WriteUInt32LE(client.ConnectionID())
	rmcResponseStream.WriteString(publicStation.EncodeToString())

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

	return 0
}
