package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// DeleteAllMessages deletes all messages for the given recipient
func DeleteAllMessages(manager *common_globals.MessagingManager, recipientID types.UInt64, recipientType messaging_constants.RecipientType) (types.UInt32, *nex.Error) {
	var err error

	result, err := manager.Database.Exec(`UPDATE FROM messaging.messages SET deleted = true WHERE
		recipient_id = $1 AND
		recipient_type = $2 AND
		reception_time + lifetime * INTERVAL '1 second' > NOW() AND
		read IS NOT TRUE AND
		deleted IS NOT TRUE
	`, recipientID, recipientType)
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	uiNbDeletedMessages, err := result.RowsAffected()
	if err != nil {
		return 0, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return types.UInt32(uiNbDeletedMessages), nil
}
