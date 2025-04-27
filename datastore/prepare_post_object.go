package datastore

import (
	"fmt"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) preparePostObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStorePreparePostParam) (*nex.RMCMessage, *nex.Error) {
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

	notUseFileServer := (param.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	if notUseFileServer {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "PreparePostObject cannot be used with DataFlagNotUseFileServer")
	}

	dataID, errCode := database.InsertObjectByPreparePostParam(manager, connection.PID(), param)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on object init: %s", errCode.Error())
		return nil, errCode
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	errCode = database.InsertObjectRatingSettings(manager, dataID, param.RatingInitParams)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on rating init: %s", errCode.Error())
		return nil, errCode
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	if param.PersistenceInitParam.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		slot := param.PersistenceInitParam.PersistenceSlotID
		oldDataID, deleteObject, err := database.GetPerpetuatedObject(manager, connection.PID(), slot)
		if err != nil {
			common_globals.Logger.Errorf("Error on persisting object: %s", err.Error())
			return nil, err
		}

		if oldDataID != datastore_constants.InvalidDataID {
			err := database.UnperpetuateObjectByDataID(manager, oldDataID, deleteObject)
			if err != nil {
				common_globals.Logger.Errorf("Error on unperpetuating object: %s", err.Error())
				return nil, err
			}
		}

		err = database.PerpetuateObject(manager, connection.PID(), param.PersistenceInitParam, dataID)
		if err != nil {
			common_globals.Logger.Errorf("Error on perpetuating object: %s", err.Error())
			return nil, err
		}
	}

	// * Format "DataID_Version", where "Version" always starts at 1
	key := fmt.Sprintf("%d_1.bin", dataID)
	postData, err := manager.S3.PresignPost(key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to sign post request")
	}

	pReqPostInfo := datastore_types.NewDataStoreReqPostInfo()

	pReqPostInfo.DataID = types.NewUInt64(dataID)
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
	rmcResponse.MethodID = datastore.MethodPreparePostObject
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterPreparePostObject != nil {
		go commonProtocol.OnAfterPreparePostObject(packet, param)
	}

	return rmcResponse, nil
}
