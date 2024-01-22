package ticket_granting

import (
	"strconv"
	"strings"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"
)

func login(err error, packet nex.PacketInterface, callID uint32, strUserName *types.String) (*nex.RMCMessage, uint32) {
	if !commonProtocol.allowInsecureLoginMethod {
		return nil, nex.Errors.Authentication.ValidationFailed
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server
	username := strUserName.Value
	var userPID *types.PID

	// TODO - This needs to change to support QRV clients, who may not send PIDs as usernames
	if username == "guest" {
		userPID = types.NewPID(100)
	} else {
		converted, err := strconv.Atoi(strings.TrimRight(username, "\x00"))
		if err != nil {
			panic(err)
		}

		userPID = types.NewPID(uint64(converted))
	}

	targetPID := types.NewPID(2) // * "Quazal Rendez-Vous" (the server user) account PID

	encryptedTicket, errorCode := generateTicket(userPID, targetPID)

	if errorCode != 0 && errorCode != nex.Errors.RendezVous.InvalidUsername {
		// * Some other error happened
		return nil, errorCode
	}

	var retval *types.QResult
	pidPrincipal := types.NewPID(0)
	pbufResponse := types.NewBuffer([]byte{})
	pConnectionData := types.NewRVConnectionData()
	strReturnMsg := types.NewString("")

	// * From the wiki:
	// *
	// * "If the username does not exist, the %retval% field is set to
	// * RendezVous::InvalidUsername and the other fields are left blank."
	if errorCode == nex.Errors.RendezVous.InvalidUsername {
		retval = types.NewQResultError(errorCode)
	} else {
		retval = types.NewQResultSuccess(nex.Errors.Core.Unknown)
		pidPrincipal = userPID
		pbufResponse = types.NewBuffer(encryptedTicket)
		strReturnMsg = commonProtocol.BuildName.Copy().(*types.String)

		specialProtocols := types.NewList[*types.PrimitiveU8]()

		specialProtocols.Type = types.NewPrimitiveU8(0)
		specialProtocols.SetFromData(commonProtocol.SpecialProtocols)

		pConnectionData.StationURL = commonProtocol.SecureStationURL
		pConnectionData.SpecialProtocols = specialProtocols
		pConnectionData.StationURLSpecialProtocols = commonProtocol.StationURLSpecialProtocols
		pConnectionData.Time = types.NewDateTime(0).Now()
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)
	pidPrincipal.WriteTo(rmcResponseStream)
	pbufResponse.WriteTo(rmcResponseStream)
	pConnectionData.WriteTo(rmcResponseStream)
	strReturnMsg.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLogin
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
