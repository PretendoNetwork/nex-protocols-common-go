package authentication

import (
	"strconv"
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-protocols-go/authentication"
)

func loginEx(err error, client *nex.Client, callID uint32, username string, oExtraData *nex.DataHolder) {
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

	rmcResponse := nex.NewRMCResponse(authentication.ProtocolID, callID)

	if errorCode != 0 && errorCode != nex.Errors.RendezVous.InvalidUsername {
		// Some other error happened
		rmcResponse.SetError(errorCode)
	} else {
		var retval *nex.Result
		var pidPrincipal uint32
		var pbufResponse []byte
		var pConnectionData *nex.RVConnectionData
		var strReturnMsg string

		pConnectionData = nex.NewRVConnectionData()
		pConnectionData.SetStationURL(commonAuthenticationProtocol.secureStationURL.EncodeToString())
		pConnectionData.SetSpecialProtocols([]byte{})
		pConnectionData.SetStationURLSpecialProtocols("")
		serverTime := nex.NewDateTime(0)
		pConnectionData.SetTime(serverTime.UTC())

		/*
			From the wiki:

			"If the username does not exist, the %retval% field is set to
			RendezVous::InvalidUsername and the other fields are left blank."
		*/
		if errorCode == nex.Errors.RendezVous.InvalidUsername {
			retval = nex.NewResultError(errorCode)
		} else {
			retval = nex.NewResultSuccess(nex.Errors.Core.Unknown)
			pidPrincipal = userPID
			pbufResponse = encryptedTicket
			strReturnMsg = commonAuthenticationProtocol.buildName
		}

		rmcResponseStream := nex.NewStreamOut(commonAuthenticationProtocol.server)

		rmcResponseStream.WriteResult(retval)
		rmcResponseStream.WriteUInt32LE(pidPrincipal)
		rmcResponseStream.WriteBuffer(pbufResponse)
		rmcResponseStream.WriteStructure(pConnectionData)
		rmcResponseStream.WriteString(strReturnMsg)

		rmcResponseBody := rmcResponseStream.Bytes()

		rmcResponse.SetSuccess(authentication.MethodLoginEx, rmcResponseBody)
	}

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonAuthenticationProtocol.server.PRUDPVersion() == 0 {
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
