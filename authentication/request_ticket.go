package authentication

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func requestTicket(err error, client *nex.Client, callID uint32, userPID uint32, serverPID uint32) {
	encryptedTicket, errorCode := generateTicket(userPID, serverPID)

	rmcResponse := nex.NewRMCResponse(nexproto.AuthenticationProtocolID, callID)

	if errorCode != 0 {
		rmcResponse.SetError(errorCode)
	} else {
		rmcResponseStream := nex.NewStreamOut(commonAuthenticationProtocol.server)

		rmcResponseStream.WriteUInt32LE(0x10001)
		rmcResponseStream.WriteBuffer(encryptedTicket)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse.SetSuccess(nexproto.AuthenticationMethodLogin, rmcResponseBody)
	}

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonAuthenticationProtocol.server.PrudpVersion() == 0 {
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

	commonAuthenticationProtocol.server.Send(responsePacket)
}
