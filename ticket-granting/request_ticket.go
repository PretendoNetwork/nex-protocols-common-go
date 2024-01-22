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
		return nil, nex.Errors.Core.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	encryptedTicket, errorCode := generateTicket(idSource, idTarget)

	if errorCode != 0 && errorCode != nex.Errors.Core.AccessDenied {
		return nil, errorCode
	}

	// * From the wiki:
	// *
	// * "If the source or target pid is invalid, the %retval% field is set to Core::AccessDenied and the ticket is empty."
	retval := types.NewQResultSuccess(nex.Errors.Core.Unknown)
	bufResponse := types.NewBuffer(encryptedTicket)

	if errorCode != 0 {
		retval = types.NewQResultError(nex.Errors.Core.AccessDenied)
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
