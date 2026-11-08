package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
)

// InsertMessage inserts a new message into the database
func InsertMessage(manager *common_globals.MessagingManager, message messaging_types.UserMessage, recipientID types.UInt64, recipientType messaging_constants.RecipientType, messageType string) (uint32, *nex.Error) {
	var err error
	var messageID uint32

	err = manager.Database.QueryRow(`INSERT INTO messaging.messages (
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
		$10
	) RETURNING id`,
		recipientID,
		recipientType,
		message.UIParentID,
		message.PIDSender,
		message.Receptiontime,
		message.UILifeTime,
		message.UIFlags,
		message.StrSubject,
		message.StrSender,
		messageType,
	).Scan(&messageID)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return messageID, nil
}
