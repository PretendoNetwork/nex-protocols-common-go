package datastore

import (
	"fmt"
	"time"

	nex "github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) prepareGetObjectV1(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStorePrepareGetParamV1) (*nex.RMCMessage, *nex.Error) {
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

	if param.DataID == types.UInt32(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * V1 has no persistence target
	metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, types.UInt64(param.DataID))
	if errCode != nil {
		return nil, errCode
	}

	// * V1 has no access password
	errCode = manager.VerifyObjectAccessPermission(*manager, connection.PID(), metaInfo, accessPassword, 0)
	if errCode != nil {
		return nil, errCode
	}

	// * The owner of an object can always view their objects, but normal users cannot
	if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
		if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
		}

		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	// TODO - Check param.LockID. See InsertObjectByPreparePostParam for notes on read locks

	notUseFileServer := (metaInfo.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	if notUseFileServer {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "PrepareGetObjectV1 cannot be used with DataFlagNotUseFileServer")
	}

	errCode = database.UpdateObjectReferenceData(manager, metaInfo.DataID)
	if errCode != nil {
		return nil, errCode
	}

	version, errCode := database.GetObjectLatestVersionNumber(manager, metaInfo.DataID)
	if errCode != nil {
		return nil, errCode
	}

	key := fmt.Sprintf("%020d_%010d.bin", metaInfo.DataID, version)
	getData, err := manager.S3.PresignGet(key, time.Minute*15)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "Failed to sign post request")
	}

	pReqGetInfo := datastore_types.NewDataStoreReqGetInfoV1()

	pReqGetInfo.URL = types.NewString(getData.URL.String())
	pReqGetInfo.RequestHeaders = types.NewList[datastore_types.DataStoreKeyValue]()
	pReqGetInfo.Size = metaInfo.Size
	pReqGetInfo.RootCACert = types.NewBuffer(getData.RootCACert)

	for key, value := range getData.RequestHeaders {
		header := datastore_types.NewDataStoreKeyValue()
		header.Key = types.NewString(key)
		header.Value = types.NewString(value)

		pReqGetInfo.RequestHeaders = append(pReqGetInfo.RequestHeaders, header)
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pReqGetInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPrepareGetObjectV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
