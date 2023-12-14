package ticket_granting

import (
	"strconv"
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func loginEx(err error, packet nex.PacketInterface, callID uint32, username string, oExtraData *nex.DataHolder) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	userPID := nex.NewPID[uint64](0)

	// TODO - This needs to change to support QRV clients, who may not send PIDs as usernames
	if username == "guest" {
		userPID = nex.NewPID[uint64](100)
	} else {
		converted, err := strconv.Atoi(strings.TrimRight(username, "\x00"))
		if err != nil {
			panic(err)
		}

		if commonTicketGrantingProtocol.server.LibraryVersion().GreaterOrEqual("4.0.0") {
			userPID = nex.NewPID[uint64](uint64(converted))
		} else {
			userPID = nex.NewPID[uint32](uint32(converted))
		}
	}

	targetPID := nex.NewPID[uint64](2) // * "Quazal Rendez-Vous" (the server user) account PID

	encryptedTicket, errorCode := generateTicket(userPID, targetPID)

	if errorCode != 0 && errorCode != nex.Errors.RendezVous.InvalidUsername {
		return nil, errorCode
	}

	var retval *nex.Result
	pidPrincipal := nex.NewPID[uint64](0)
	var pbufResponse []byte
	var pConnectionData *nex.RVConnectionData
	var strReturnMsg string

	pConnectionData = nex.NewRVConnectionData()
	pConnectionData.StationURL = commonTicketGrantingProtocol.SecureStationURL
	pConnectionData.SpecialProtocols = []byte{}
	pConnectionData.StationURLSpecialProtocols = nex.NewStationURL("")
	pConnectionData.Time = nex.NewDateTime(0).Now()

	// * From the wiki:
	// *
	// * "If the username does not exist, the %retval% field is set to
	// * RendezVous::InvalidUsername and the other fields are left blank."
	if errorCode == nex.Errors.RendezVous.InvalidUsername {
		retval = nex.NewResultError(errorCode)
	} else {
		retval = nex.NewResultSuccess(nex.Errors.Core.Unknown)
		pidPrincipal = userPID
		pbufResponse = encryptedTicket
		strReturnMsg = commonTicketGrantingProtocol.BuildName
	}

	rmcResponseStream := nex.NewStreamOut(commonTicketGrantingProtocol.server)

	rmcResponseStream.WriteResult(retval)
	rmcResponseStream.WritePID(pidPrincipal)
	rmcResponseStream.WriteBuffer(pbufResponse)
	rmcResponseStream.WriteStructure(pConnectionData)
	rmcResponseStream.WriteString(strReturnMsg)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLoginEx
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
