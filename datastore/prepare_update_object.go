package datastore

import (
	"fmt"
	"math"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) prepareUpdateObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStorePrepareUpdateParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager

	if manager.S3 == nil {
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "S3 config not set")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// TODO - Add rollback for when error occurs

	if param.DataID == types.UInt64(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// TODO - Move this to VerifyObjectUpdatePermission?
	// * Objects in the DataID range 900,000-999,999 are special
	if param.DataID < 1000000 {
		// * Unsure if this is the correct error, but it feels right
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
	if errCode != nil {
		return nil, errCode
	}

	// * If the object is pending or rejected, only the owner can interact with it
	if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) || metaInfo.Status == types.UInt8(datastore_constants.DataStatusRejected)) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	notUseFileServer := (metaInfo.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	if notUseFileServer {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "PrepareUpdateObject cannot be used with DataFlagNotUseFileServer")
	}

	newVersion, errCode := database.UpdateObjectByPrepareUpdateParam(manager, param, metaInfo)
	if errCode != nil {
		return nil, errCode
	}

	// * NEX 3.0.0 bumped versions from uint16 to uint32.
	// * Older clients can't handle larger numbers
	// TODO - Check this before inserting into the database
	if endpoint.LibraryVersions().DataStore.Major < 3 {
		if newVersion > math.MaxUint16 {
			// * OverCapacity is definitely not the correct error here, but it sounds nice
			return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
		}
	}

	notifyAccessRecipientsOnUpdate := (metaInfo.Flag & types.UInt32(datastore_constants.DataFlagUseNotificationOnUpdate)) != 0
	if notifyAccessRecipientsOnUpdate {
		recipientIDs, errCode := manager.GetNotificationRecipients(metaInfo.OwnerID, metaInfo.Permission)
		if errCode != nil {
			common_globals.Logger.Errorf("Error on getting notification recipients: %s", errCode.Error())
			return nil, errCode
		}

		for _, recipientID := range recipientIDs {
			errCode := database.SendNotification(manager, uint64(metaInfo.DataID), recipientID, connection.PID())
			if errCode != nil {
				common_globals.Logger.Errorf("Error on sending notification: %s", errCode.Error())
				return nil, errCode
			}
		}
	}

	// * Format "DataID_Version"
	key := fmt.Sprintf("objects/%020d_%010d.bin", param.DataID, newVersion)
	postData, err := manager.S3.PresignPost(key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to sign post request")
	}

	pReqPostInfo := datastore_types.NewDataStoreReqUpdateInfo()

	pReqPostInfo.Version = newVersion
	pReqPostInfo.URL = types.NewString(postData.URL.String())
	pReqPostInfo.RequestHeaders = types.NewList[datastore_types.DataStoreKeyValue]()
	pReqPostInfo.FormFields = types.NewList[datastore_types.DataStoreKeyValue]()
	pReqPostInfo.RootCACert = types.NewBuffer(postData.RootCACert)

	for key, value := range postData.RequestHeaders {
		header := datastore_types.NewDataStoreKeyValue()
		header.Key = types.NewString(key)
		header.Value = types.NewString(value)

		pReqPostInfo.RequestHeaders = append(pReqPostInfo.RequestHeaders, header)
	}

	for key, value := range postData.FormData {
		field := datastore_types.NewDataStoreKeyValue()
		field.Key = types.NewString(key)
		field.Value = types.NewString(value)

		pReqPostInfo.FormFields = append(pReqPostInfo.FormFields, field)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pReqPostInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPrepareUpdateObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
