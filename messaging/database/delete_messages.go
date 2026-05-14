package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// DeleteMessages deletes the given messages for the given recipient
func DeleteMessages(manager *common_globals.MessagingManager, recipientID types.UInt64, recipientType messaging_constants.RecipientType, lstMessagesToDelete types.List[types.UInt32]) *nex.Error {
	var err error

	_, err = manager.Database.Exec(`UPDATE FROM messaging.messages SET deleted = true WHERE
		id = ANY($1) AND
		recipient_id = $2 AND
		recipient_type = $3 AND
		reception_time + lifetime * INTERVAL '1 second' > NOW() AND
		read IS NOT TRUE AND
		deleted IS NOT TRUE
	`, lstMessagesToDelete, recipientID, recipientType)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	return nil
}
