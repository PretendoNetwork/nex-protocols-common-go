package messaging

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"

	messaging "github.com/PretendoNetwork/nex-protocols-go/v2/messaging"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	messaging_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/messaging/database"
)

func (commonProtocol *CommonProtocol) retrieveMessages(err error, packet nex.PacketInterface, callID uint32, recipient messaging_types.MessageRecipient, lstMsgIDs types.List[types.UInt32], bLeaveOnServer types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	if len(lstMsgIDs) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "lstMsgIDs must not be empty")
	}

	// * Only allow up to 100 messages
	if len(lstMsgIDs) > 100 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Trying to query over 100 messages")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	libraryVersion := server.LibraryVersions.Messaging

	var lstMessages types.List[types.DataHolder]
	var nexError *nex.Error

	recipientID, recipientType := common_globals.GetMessageRecipientData(libraryVersion, recipient)

	// * If the MessageRecipient is invalid, just return no entries
	if valid := commonProtocol.manager.ValidateMessageRecipient(commonProtocol.manager, connection.PID(), recipientID, recipientType); valid {
		lstMessages, nexError = messaging_database.RetrieveMessages(commonProtocol.manager, recipientID, recipientType, lstMsgIDs, bLeaveOnServer)
		if nexError != nil {
			return nil, nexError
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	lstMessages.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = messaging.ProtocolID
	rmcResponse.MethodID = messaging.MethodRetrieveMessages
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
