package messaging

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"

	messaging "github.com/PretendoNetwork/nex-protocols-go/v2/messaging"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	messaging_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/messaging/database"
)

func (commonProtocol *CommonProtocol) getNumberOfMessages(err error, packet nex.PacketInterface, callID uint32, recipient messaging_types.MessageRecipient) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	libraryVersion := server.LibraryVersions.Messaging

	var uiNbMessages types.UInt32 = 0
	var nexError *nex.Error

	recipientID, recipientType := common_globals.GetMessageRecipientData(libraryVersion, recipient)

	// * If the MessageRecipient is invalid, just return 0
	if valid := commonProtocol.manager.ValidateMessageRecipient(commonProtocol.manager, connection.PID(), recipientID, recipientType); valid {
		uiNbMessages, nexError = messaging_database.GetNumberOfMessages(commonProtocol.manager, recipientID, recipientType)
		if nexError != nil {
			return nil, nexError
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	uiNbMessages.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = messaging.ProtocolID
	rmcResponse.MethodID = messaging.MethodGetNumberOfMessages
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
