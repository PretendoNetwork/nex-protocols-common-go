package database

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func SendNotification(manager *common_globals.DataStoreManager, dataID uint64, recipientID, senderPID types.PID) *nex.Error {
	var notificationID uint64

	err := manager.Database.QueryRow(`INSERT INTO datastore.notifications (
		data_id,
		recipient_id,
		sender_pid
	) VALUES (
		$1,
		$2,
		$3
	) RETURNING id`, dataID, recipientID, senderPID).Scan(&notificationID)

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	key := fmt.Sprintf("%s/notifications/%020d.bin", manager.S3.KeyBase, recipientID)
	data := fmt.Sprintf("%d,%d,%d", notificationID, recipientID, manager.NotifyTimestamp)
	err = manager.S3.Manager.PutObject(manager.S3.Bucket, key, data)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to put notifier")
	}

	return nil
}
