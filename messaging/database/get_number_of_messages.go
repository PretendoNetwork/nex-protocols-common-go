package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// GetNumberOfMessages gets the number of messages available for the given recipient
func GetNumberOfMessages(manager *common_globals.MessagingManager, recipientID types.UInt64, recipientType messaging_constants.RecipientType) (types.UInt32, *nex.Error) {
	var err error
	var uiNbMessages types.UInt32

	err = manager.Database.QueryRow(`SELECT COUNT(id) FROM messaging.messages WHERE
		recipient_id = $1 AND
		recipient_type = $2 AND
		reception_time + lifetime * INTERVAL '1 second' > NOW() AND
		read IS NOT TRUE AND
		deleted IS NOT TRUE
	`, recipientID, recipientType).Scan(&uiNbMessages)
	if err != nil {
		return uiNbMessages, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return uiNbMessages, nil
}
