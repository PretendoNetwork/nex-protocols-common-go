package messaging

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"

	messaging "github.com/PretendoNetwork/nex-protocols-go/v2/messaging"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	messaging_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/messaging/database"
)

func (commonProtocol *CommonProtocol) deleteMessages(err error, packet nex.PacketInterface, callID uint32, recipient messaging_types.MessageRecipient, lstMessagesToDelete types.List[types.UInt32]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	if len(lstMessagesToDelete) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "lstMessagesToDelete must not be empty")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)
	server := endpoint.Server

	libraryVersion := server.LibraryVersions.Messaging

	var nexError *nex.Error

	recipientID, recipientType := common_globals.GetMessageRecipientData(libraryVersion, recipient)

	// * If the MessageRecipient is invalid, just do nothing
	if valid := commonProtocol.manager.ValidateMessageRecipient(commonProtocol.manager, connection.PID(), recipientID, recipientType); valid {
		nexError = messaging_database.DeleteMessages(commonProtocol.manager, recipientID, recipientType, lstMessagesToDelete)
		if nexError != nil {
			return nil, nexError
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = messaging.ProtocolID
	rmcResponse.MethodID = messaging.MethodDeleteMessages
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
