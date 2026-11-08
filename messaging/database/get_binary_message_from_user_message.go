package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetBinaryMessageFromUserMessage gets the binary body for a given user message
func GetBinaryMessageFromUserMessage(manager *common_globals.MessagingManager, messageHeader messaging_types.UserMessage) (messaging_types.BinaryMessage, *nex.Error) {
	var binaryMessage messaging_types.BinaryMessage
	var err error

	err = manager.Database.QueryRow(`SELECT body FROM messaging.binary_messages WHERE id = $1`, messageHeader.UIID).Scan(&binaryMessage.BinaryBody)
	if err != nil {
		return binaryMessage, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	binaryMessage.UserMessage = messageHeader

	return binaryMessage, nil
}
