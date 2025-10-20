package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getNewArrivedNotficationsV1(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetNewArrivedNotificationsParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if uint32(param.Limit) > datastore_constants.MaxSearchResultSize {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pResultRaw, pHasNext, errCode := database.GetNewArrivedNotifications(manager, connection.PID(), param)
	if errCode != nil {
		return nil, errCode
	}

	// * Convert types.List[datastore_types.DataStoreNotification] to types.List[datastore_types.DataStoreNotificationV1]
	// TODO - Handle case where the data ID is bigger than the maximum supported on a uint32
	pResult := make(types.List[datastore_types.DataStoreNotificationV1], len(pResultRaw))
	for i, notification := range pResultRaw {
		pResult[i].NotificationID = notification.NotificationID
		pResult[i].DataID = types.UInt32(notification.DataID)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pResult.WriteTo(rmcResponseStream)
	pHasNext.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetNewArrivedNotificationsV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}

