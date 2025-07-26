package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetNewArrivedNotifications(manager *common_globals.DataStoreManager, recipientID types.PID, param datastore_types.DataStoreGetNewArrivedNotificationsParam) (types.List[datastore_types.DataStoreNotification], types.Bool, *nex.Error) {
	// * First we mark all unread notifications as read
	_, err := manager.Database.Exec(`UPDATE datastore.notifications SET read = true, read_date = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') WHERE recipient_id = $1 AND id <= $2 AND read IS NOT TRUE`, recipientID, param.LastNotificationID)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	var pResult types.List[datastore_types.DataStoreNotification]
	var pHasNext types.Bool = false

	// * Get the full count so we can determine if we are giving all notifications
	var notificationsCount uint64
	err = manager.Database.QueryRow(`SELECT COUNT(id) FROM datastore.notifications WHERE recipient_id = $1 AND read IS NOT TRUE`, recipientID).Scan(&notificationsCount)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	rows, err := manager.Database.Query(`SELECT id, data_id FROM datastore.notifications WHERE recipient_id = $1 AND read IS NOT TRUE LIMIT $2`, recipientID, param.Limit)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	for rows.Next() {
		var notification datastore_types.DataStoreNotification
		err = rows.Scan(&notification.NotificationID, &notification.DataID)
		if err != nil {
			// TODO - Send more specific errors?
			return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		pResult = append(pResult, notification)
	}

	rows.Close()

	if notificationsCount > uint64(len(pResult)) {
		pHasNext = true
	}

	return pResult, pHasNext, nil
}
