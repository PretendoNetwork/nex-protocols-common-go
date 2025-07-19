package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

func (commonProtocol *CommonProtocol) login(err error, packet nex.PacketInterface, callID uint32, strUserName types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	var errorCode *nex.Error

	if !commonProtocol.allowInsecureLoginMethod {
		common_globals.Logger.Error("TicketGranting::Login blocked")
		errorCode = nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "TicketGranting::Login blocked")
	}

	var sourceAccount *nex.Account
	if errorCode == nil {
		sourceAccount, errorCode = endpoint.AccountDetailsByUsername(string(strUserName))
	}

	var targetAccount *nex.Account
	if errorCode == nil {
		targetAccount, errorCode = endpoint.AccountDetailsByUsername(commonProtocol.SecureServerAccount.Username)
	}

	if errorCode == nil && sourceAccount.RequiresTokenAuth {
		common_globals.Logger.Error("Source account requires token authentication")
		errorCode = nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Source account requires token authentication")
	}

	var encryptedTicket []byte
	if errorCode == nil {
		encryptedTicket, errorCode = generateTicket(sourceAccount, targetAccount, nil, commonProtocol.SessionKeyLength, endpoint)
	}

	var retval types.QResult
	pidPrincipal := types.NewPID(0)
	pbufResponse := types.NewBuffer([]byte{})
	pConnectionData := types.NewRVConnectionData()
	strReturnMsg := types.NewString("")

	// * If any errors are triggered, return them in %retval%
	if errorCode != nil {
		retval = types.NewQResultError(errorCode.ResultCode)
	} else {
		retval = types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
		pidPrincipal = sourceAccount.PID
		pbufResponse = types.NewBuffer(encryptedTicket)
		strReturnMsg = commonProtocol.BuildName.Copy().(types.String)

		specialProtocols := types.List[types.UInt8](commonProtocol.SpecialProtocols)

		pConnectionData.StationURL = commonProtocol.SecureStationURL
		pConnectionData.SpecialProtocols = specialProtocols
		pConnectionData.StationURLSpecialProtocols = commonProtocol.StationURLSpecialProtocols
		pConnectionData.Time = types.NewDateTime(0).Now()

		if server.LibraryVersions.Main.GreaterOrEqual("3.5.0") {
			pConnectionData.StructureVersion = 1
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	pidPrincipal.WriteTo(rmcResponseStream)
	pbufResponse.WriteTo(rmcResponseStream)
	pConnectionData.WriteTo(rmcResponseStream)
	strReturnMsg.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLogin
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterLogin != nil {
		go commonProtocol.OnAfterLogin(packet, strUserName)
	}

	return rmcResponse, nil
}
