package ticket_granting

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"
)

func requestTicket(err error, packet nex.PacketInterface, callID uint32, idSource *types.PID, idTarget *types.PID) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	sourceAccount, errorCode := endpoint.AccountDetailsByPID(idSource)
	if errorCode != nil && errorCode.ResultCode != nex.ResultCodes.Core.AccessDenied {
		return nil, errorCode.ResultCode
	}

	targetAccount, errorCode := endpoint.AccountDetailsByPID(idTarget)
	if errorCode != nil && errorCode.ResultCode != nex.ResultCodes.Core.AccessDenied {
		return nil, errorCode.ResultCode
	}

	encryptedTicket, errorCode := generateTicket(sourceAccount, targetAccount, commonProtocol.SessionKeyLength, server)
	if errorCode != nil && errorCode.ResultCode != nex.ResultCodes.Core.AccessDenied {
		return nil, errorCode.ResultCode
	}

	// * From the wiki:
	// *
	// * "If the source or target pid is invalid, the %retval% field is set to Core::AccessDenied and the ticket is empty."
	retval := types.NewQResultSuccess(nex.ResultCodes.Core.Unknown)
	bufResponse := types.NewBuffer(encryptedTicket)

	if errorCode != nil {
		retval = types.NewQResultError(nex.ResultCodes.Core.AccessDenied)
		bufResponse = types.NewBuffer([]byte{})
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)
	bufResponse.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodRequestTicket
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
