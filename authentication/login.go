package authentication

import (
	"strconv"
	"strings"

	"github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func login(err error, client *nex.Client, callID uint32, username string) {
	var userPID uint32

	if username == "guest" {
		userPID = 100
	} else {
		converted, err := strconv.Atoi(strings.TrimRight(username, "\x00"))
		if err != nil {
			panic(err)
		}

		userPID = uint32(converted)
	}

	var targetPID uint32 = 2 // "Quazal Rendez-Vous" (the server user) account PID

	encryptedTicket, errorCode := generateTicket(userPID, targetPID)

	rmcResponse := nex.NewRMCResponse(nexproto.AuthenticationProtocolID, callID)

	if errorCode != 0 {
		rmcResponse.SetError(errorCode)
	} else {
		rvConnectionData := nex.NewRVConnectionData()
		rvConnectionData.SetStationURL(commonAuthenticationProtocol.secureStationURL.EncodeToString())
		rvConnectionData.SetSpecialProtocols([]byte{})
		rvConnectionData.SetStationURLSpecialProtocols("")

		rmcResponseStream := nex.NewStreamOut(commonAuthenticationProtocol.server)

		rmcResponseStream.WriteUInt32LE(0x10001)
		rmcResponseStream.WriteUInt32LE(uint32(userPID))
		rmcResponseStream.WriteBuffer(encryptedTicket)
		rmcResponseStream.WriteStructure(rvConnectionData)
		rmcResponseStream.WriteString(commonAuthenticationProtocol.buildName)

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
