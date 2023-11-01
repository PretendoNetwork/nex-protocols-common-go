package secureconnection

import (
	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func replaceURL(err error, packet nex.PacketInterface, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonSecureConnectionProtocol.server
	client := packet.Sender()

	stations := client.StationURLs()
	for i := 0; i < len(stations); i++ {
		currentStation := stations[i]
		if currentStation.Address() == oldStation.Address() && currentStation.Port() == oldStation.Port() {
			newStation.SetPID(client.PID()) //This fixes Minecraft, but is obviously incorrect. TODO: What are we really meant to do here?
			stations[i] = newStation
		}
	}

	rmcResponse := nex.NewRMCResponse(secure_connection.ProtocolID, callID)
	rmcResponse.SetSuccess(secure_connection.MethodReplaceURL, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
