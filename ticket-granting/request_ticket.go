package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
)

func (commonProtocol *CommonProtocol) requestTicket(err error, packet nex.PacketInterface, callID uint32, idSource types.PID, idTarget types.PID) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	var errorCode *nex.Error

	sourceAccount, errorCode := endpoint.AccountDetailsByPID(idSource)

	var targetAccount *nex.Account
	if errorCode == nil {
		targetAccount, errorCode = endpoint.AccountDetailsByPID(idTarget)
	}

	if errorCode == nil && sourceAccount.RequiresTokenAuth {
		common_globals.Logger.Error("Source account requires token authentication")
		errorCode = nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Source account requires token authentication")
	}

	var encryptedTicket []byte
	if errorCode == nil {
		encryptedTicket, errorCode = generateTicket(sourceAccount, targetAccount, nil, commonProtocol.SessionKeyLength, endpoint)
	}

	// * If any errors are triggered, return them in %retval%
	retval := types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
	bufResponse := types.NewBuffer(encryptedTicket)
	pSourceKey := types.NewString("")

	if errorCode != nil {
		retval = types.NewQResultError(errorCode.ResultCode)
		bufResponse = types.NewBuffer([]byte{})
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	bufResponse.WriteTo(rmcResponseStream)

	if server.LibraryVersions.Main.GreaterOrEqual("4.0.0") {
		pSourceKey.WriteTo(rmcResponseStream)
	}

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodRequestTicket
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterRequestTicket != nil {
		go commonProtocol.OnAfterRequestTicket(packet, idSource, idTarget)
	}

	return rmcResponse, nil
}
