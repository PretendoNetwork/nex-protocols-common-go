package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func GetNewArrivedNotifications(database common_globals.Database, recipientID types.PID, param datastore_types.DataStoreGetNewArrivedNotificationsParam) (types.List[datastore_types.DataStoreNotification], types.Bool, *nex.Error) {
	// * First we mark all unread notifications as read
	_, err := database.Exec(`UPDATE datastore.notifications SET read = true, read_date = (CURRENT_TIMESTAMP AT TIME ZONE 'UTC') WHERE recipient_id = $1 AND id <= $2 AND read IS NOT TRUE`, recipientID, param.LastNotificationID)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	var pResult types.List[datastore_types.DataStoreNotification]
	var pHasNext types.Bool = false

	limit := param.Limit
	if limit == 0 || limit >= 100 {
		limit = 20
	}

	// * Add 1 to the limit to check for more notifications remaining.
	// * Notifications for deleted objects are skipped
	// TODO - Hard delete notifications when the object is deleted rather than skipping?
	rows, err := database.Query(`
		SELECT n.id, n.data_id
		FROM datastore.notifications n
		WHERE n.recipient_id = $1 AND n.read IS NOT TRUE AND NOT EXISTS (
			SELECT 1 FROM datastore.objects o WHERE o.data_id = n.data_id AND o.deleted = true
		)
		ORDER BY n.id
		LIMIT $2+1
	`, recipientID, limit)
	if err != nil {
		// TODO - Send more specific errors?
		return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		if len(pResult) == int(limit) {
			pHasNext = true
			break
		}

		var notification datastore_types.DataStoreNotification
		err = rows.Scan(&notification.NotificationID, &notification.DataID)
		if err != nil {
			// TODO - Send more specific errors?
			return nil, false, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		pResult = append(pResult, notification)
	}

	return pResult, pHasNext, nil
}
