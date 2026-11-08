package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func SendUpdateNotifications(database common_globals.Database, manager *common_globals.DataStoreManager, dataID types.UInt64, senderPID types.PID) *nex.Error {
	return sendObjectNotifications(database, manager, dataID, datastore_constants.DataFlagUseNotificationOnUpdate, senderPID)
}
