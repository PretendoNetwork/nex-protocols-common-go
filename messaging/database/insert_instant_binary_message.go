package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// InsertInstantBinaryMessage inserts a new instant binary message into the database
func InsertInstantBinaryMessage(manager *common_globals.MessagingManager, message messaging_types.BinaryMessage, recipientID types.UInt64, recipientType types.UInt32) *nex.Error {
	var err error

	messageID, nexError := InsertInstantMessage(manager, message.UserMessage, recipientID, recipientType, "BinaryMessage")
	if nexError != nil {
		return nexError
	}

	_, err = manager.Database.Exec(`INSERT INTO messaging.instant_binary_messages (
		id,
		body
	) VALUES (
		$1,
		$2
	)`, messageID, message.BinaryBody)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
