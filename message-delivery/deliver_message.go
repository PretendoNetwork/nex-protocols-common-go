package message_delivery

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"

	message_delivery "github.com/PretendoNetwork/nex-protocols-go/v2/message-delivery"
)

func (commonProtocol *CommonProtocol) deliverMessage(err error, packet nex.PacketInterface, callID uint32, oUserMessage types.DataHolder) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// NOTE - This method will silently fail on any errors
	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = message_delivery.ProtocolID
	rmcResponse.MethodID = message_delivery.MethodDeliverMessage
	rmcResponse.CallID = callID

	recipientID, recipientType, nexError := commonProtocol.manager.ValidateMessage(oUserMessage)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return rmcResponse, nil
	}

	oUserMessage, nexError = commonProtocol.manager.PrepareMessage(connection.PID(), oUserMessage)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return rmcResponse, nil
	}

	_, _, _, nexError = commonProtocol.manager.ProcessMessage(oUserMessage, recipientID, recipientType, true)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return rmcResponse, nil
	}

	if commonProtocol.OnAfterDeliverMessage != nil {
		go commonProtocol.OnAfterDeliverMessage(packet, oUserMessage)
	}

	return rmcResponse, nil
}
