package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
)

// InsertInstantTextMessage inserts a new instant text message into the database
func InsertInstantTextMessage(manager *common_globals.MessagingManager, message messaging_types.TextMessage, recipientID types.UInt64, recipientType messaging_constants.RecipientType) *nex.Error {
	var err error

	_, err = manager.Database.Exec(`WITH message_id AS (INSERT INTO messaging.instant_messages (
		recipient_id,
		recipient_type,
		parent_id,
		sender_pid,
		reception_time,
		lifetime,
		flags,
		subject,
		sender,
		type
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$8,
		$9,
		'TextMessage'
	) RETURNING id) INSERT INTO messaging.instant_text_messages (
		id,
		body
	) VALUES (
		(SELECT id FROM message_id),
		$10
	)`,
		recipientID,
		recipientType,
		message.UserMessage.UIParentID,
		message.UserMessage.PIDSender,
		message.UserMessage.Receptiontime,
		message.UserMessage.UILifeTime,
		message.UserMessage.UIFlags,
		message.UserMessage.StrSubject,
		message.UserMessage.StrSender,
		message.StrTextBody,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
