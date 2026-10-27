package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func SendPostNotifications(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, senderPID types.PID) *nex.Error {
	var metaInfo datastore_types.DataStoreMetaInfo

	err := database.QueryRow(`
		SELECT
			owner,
			access_permission,
			access_permission_recipients,
			raw_flags
		FROM datastore.objects
		WHERE data_id=$1`, dataID).Scan(
		&metaInfo.OwnerID,
		&metaInfo.Permission.Permission,
		&metaInfo.Permission.RecipientIDs,
		&metaInfo.Flag,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nex.NewError(nex.ResultCodes.DataStore.NotFound, err.Error())
		}

		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if !metaInfo.Flag.HasFlag(datastore_constants.DataFlagUseNotificationOnPost) {
		return nil
	}

	recipientIDs, errCode := manager.GetNotificationRecipients(metaInfo.OwnerID, metaInfo.Permission)
	if errCode != nil {
		common_globals.Logger.Errorf("Error getting notification recipients: %s", errCode.Error())
		return errCode
	}

	for _, recipientID := range recipientIDs {
		errCode = SendNotification(database, manager, uint64(dataID), recipientID, senderPID)
		if errCode != nil {
			common_globals.Logger.Errorf("Error sending notification: %s", errCode.Error())
			return errCode
		}
	}

	return nil
}
