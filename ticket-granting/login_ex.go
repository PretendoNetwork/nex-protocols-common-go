package ticket_granting

import (
	"strconv"
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func loginEx(err error, packet nex.PacketInterface, callID uint32, username string, oExtraData *nex.DataHolder) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)

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

	if errorCode != 0 && errorCode != nex.Errors.RendezVous.InvalidUsername {
		// Some other error happened
		return errorCode
	}

	var retval *nex.Result
	var pidPrincipal uint32
	var pbufResponse []byte
	var pConnectionData *nex.RVConnectionData
	var strReturnMsg string

	pConnectionData = nex.NewRVConnectionData()
	pConnectionData.SetStationURL(commonTicketGrantingProtocol.secureStationURL.EncodeToString())
	pConnectionData.SetSpecialProtocols([]byte{})
	pConnectionData.SetStationURLSpecialProtocols("")
	serverTime := nex.NewDateTime(0)
	pConnectionData.SetTime(nex.NewDateTime(serverTime.UTC()))

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
		strReturnMsg = commonTicketGrantingProtocol.buildName
	}

	rmcResponseStream := nex.NewStreamOut(commonTicketGrantingProtocol.server)

	rmcResponseStream.WriteResult(retval)
	rmcResponseStream.WriteUInt32LE(pidPrincipal)
	rmcResponseStream.WriteBuffer(pbufResponse)
	rmcResponseStream.WriteStructure(pConnectionData)
	rmcResponseStream.WriteString(strReturnMsg)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLoginEx
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if commonTicketGrantingProtocol.server.PRUDPVersion == 0 {
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

	commonTicketGrantingProtocol.server.Send(responsePacket)

	return 0
}
