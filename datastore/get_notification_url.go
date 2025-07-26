package datastore

import (
	"fmt"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getNotificationURL(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetNotificationURLParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	key := fmt.Sprintf("%020d.bin", connection.PID())
	getData, err := manager.S3.PresignNotify(key, time.Hour*24*7) // * 1 week
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to sign get request")
	}

	url := getData.URL

	// * Replicate the same encoding as official servers
	var info datastore_types.DataStoreReqGetNotificationURLInfo
	info.URL = types.String(url.Scheme + "://" + url.Host + "/")
	info.Key = types.String(url.Path)
	info.Query = types.String("?" + url.RawQuery)

	// * This method triggers a new notification with an invalid data ID
	errCode := database.SendNotification(manager, datastore_constants.InvalidDataID, connection.PID(), connection.PID())
	if errCode != nil {
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	info.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetNotificationURL
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
