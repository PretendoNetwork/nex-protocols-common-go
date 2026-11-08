package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// RetrieveDetailedMessage gets a full message given a message header
func RetrieveDetailedMessage(manager *common_globals.MessagingManager, messageHeader messaging_types.UserMessage, messageType string) (types.DataHolder, *nex.Error) {
	var messageHolder types.DataHolder
	var message types.RVType
	var nexError *nex.Error

	switch messageType {
	case "TextMessage":
		message, nexError = GetTextMessageFromUserMessage(manager, messageHeader)
	case "BinaryMessage":
		message, nexError = GetBinaryMessageFromUserMessage(manager, messageHeader)
	default:
		return messageHolder, nex.NewError(nex.ResultCodes.Core.Exception, "Invalid message type")
	}

	if nexError != nil {
		return messageHolder, nexError
	}

	messageHolder.Object = message.Copy().(types.DataInterface)
	return messageHolder, nil
}
