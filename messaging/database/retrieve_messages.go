package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// RetrieveMessages retrieves the specified messages for the given recipient
func RetrieveMessages(manager *common_globals.MessagingManager, recipientID types.UInt64, recipientType messaging_constants.RecipientType, lstMsgIDs types.List[types.UInt32], bLeaveOnServer types.Bool) (types.List[types.DataHolder], *nex.Error) {
	lstMessages := make(types.List[types.DataHolder], 0, len(lstMsgIDs))

	rows, err := manager.Database.Query(`UPDATE messaging.messages SET read = $4 -- Use the inverse of bLeaveOnServer to not mark messages as read if set
	WHERE
		recipient_id = $1 AND
		recipient_type = $2 AND
		reception_time + lifetime * INTERVAL '1 second' > NOW() AND
		read IS NOT TRUE AND
		deleted IS NOT TRUE AND
		id = ANY($3)
	RETURNING
		id,
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
	`, recipientID, recipientType, lstMsgIDs, !bLeaveOnServer)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	defer rows.Close()

	libraryVersion := manager.Endpoint.Server.LibraryVersions.Messaging

	for rows.Next() {
		var messageHeader messaging_types.UserMessage
		var recipientID types.UInt64
		var recipientType messaging_constants.RecipientType
		var messageType string

		err = rows.Scan(
			&messageHeader.UIID,
			&recipientID,
			&recipientType,
			&messageHeader.UIParentID,
			&messageHeader.PIDSender,
			&messageHeader.Receptiontime,
			&messageHeader.UILifeTime,
			&messageHeader.UIFlags,
			&messageHeader.StrSubject,
			&messageHeader.StrSender,
			&messageType,
		)
		if err != nil {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		messageHeader = common_globals.SetUserMessageRecipientData(libraryVersion, messageHeader, recipientID, recipientType)

		message, nexError := manager.RetrieveDetailedMessage(manager, messageHeader, messageType)
		if nexError != nil {
			return nil, nexError
		}

		lstMessages = append(lstMessages, message)
	}

	return lstMessages, nil
}
