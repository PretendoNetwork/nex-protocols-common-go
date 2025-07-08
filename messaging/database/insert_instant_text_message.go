package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// InsertInstantTextMessage inserts a new instant text message into the database
func InsertInstantTextMessage(manager *common_globals.MessagingManager, message messaging_types.TextMessage, recipientID types.UInt64, recipientType types.UInt32) *nex.Error {
	var err error

	messageID, nexError := InsertInstantMessage(manager, message.UserMessage, recipientID, recipientType, "TextMessage")
	if nexError != nil {
		return nexError
	}

	_, err = manager.Database.Exec(`INSERT INTO messaging.instant_text_messages (
		id,
		body
	) VALUES (
		$1,
		$2
	)`, messageID, message.StrTextBody)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
