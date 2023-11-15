package ticket_granting

import (
	nex "github.com/PretendoNetwork/nex-go"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/ticket-granting"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func requestTicket(err error, packet nex.PacketInterface, callID uint32, userPID *nex.PID, targetPID *nex.PID) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	encryptedTicket, errorCode := generateTicket(userPID, targetPID)

	// * If the source or target pid is invalid,
	// * the %retval% field is set to Core::AccessDenied and the ticket is empty.
	if errorCode != 0 {
		return nil, errorCode
	}

	rmcResponseStream := nex.NewStreamOut(commonTicketGrantingProtocol.server)

	rmcResponseStream.WriteResult(nex.NewResultSuccess(nex.Errors.Core.Unknown))
	rmcResponseStream.WriteBuffer(encryptedTicket)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodRequestTicket
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
