package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetTextMessageFromUserMessage gets the text body for a given user message
func GetTextMessageFromUserMessage(manager *common_globals.MessagingManager, messageHeader messaging_types.UserMessage) (messaging_types.TextMessage, *nex.Error) {
	var textMessage messaging_types.TextMessage
	var err error

	err = manager.Database.QueryRow(`SELECT body FROM messaging.text_messages WHERE id = $1`, messageHeader.UIID).Scan(&textMessage.StrTextBody)
	if err != nil {
		return textMessage, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	textMessage.UserMessage = messageHeader

	return textMessage, nil
}
