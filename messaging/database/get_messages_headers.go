package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetMessagesHeaders gets the message headers available for the given recipient
func GetMessagesHeaders(manager *common_globals.MessagingManager, recipientID types.UInt64, recipientType messaging_constants.RecipientType, resultRange types.ResultRange) (types.List[messaging_types.UserMessage], *nex.Error) {
	lstMsgHeaders := make(types.List[messaging_types.UserMessage], 0, resultRange.Length)

	rows, err := manager.Database.Query(`SELECT (
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
	) FROM messaging.messages WHERE
		recipient_id = $1 AND
		recipient_type = $2 AND
		reception_time + lifetime * INTERVAL '1 second' > NOW() AND
		read IS NOT TRUE AND
		deleted IS NOT TRUE
		OFFSET $3
		LIMIT $4
	`, recipientID, recipientType, resultRange.Offset, resultRange.Length)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	defer rows.Close()

	libraryVersion := manager.Endpoint.Server.LibraryVersions.Messaging

	for rows.Next() {
		var messageHeader messaging_types.UserMessage
		var recipientID types.UInt64
		var recipientType messaging_constants.RecipientType

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
		)
		if err != nil {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		messageHeader = common_globals.SetUserMessageRecipientData(libraryVersion, messageHeader, recipientID, recipientType)
		lstMsgHeaders = append(lstMsgHeaders, messageHeader)
	}

	return lstMsgHeaders, nil
}
