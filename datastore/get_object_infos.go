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

func (commonProtocol *CommonProtocol) getObjectInfos(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager

	if manager.S3 == nil {
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "S3 config not set")
	}

	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// TODO - Add rollback for when error occurs

	pInfos := types.NewList[datastore_types.DataStoreReqGetInfo]()
	pResults := types.NewList[types.QResult]()
	invalidReqGetInfo := datastore_types.NewDataStoreReqGetInfo() // * Quick hack to get a zeroed struct

	// * param.DataID and param.PersistenceTarget are ignored here
	for _, dataID := range dataIDs {
		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, 0)
		if errCode != nil {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		// TODO - Check param.LockID. See InsertObjectByPreparePostParam for notes on read locks

		notUseFileServer := (metaInfo.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
		if notUseFileServer {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		errCode = database.UpdateObjectReferenceData(manager, metaInfo.DataID)
		if errCode != nil {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		version, errCode := database.GetObjectLatestVersionNumber(manager, metaInfo.DataID)
		if errCode != nil {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		key := fmt.Sprintf("objects/%020d_%010d.bin", metaInfo.DataID, version)
		getData, err := manager.S3.PresignGet(key, time.Minute*15)
		if err != nil {
			pInfos = append(pInfos, invalidReqGetInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.Unknown))
			continue
		}

		pReqGetInfo := datastore_types.NewDataStoreReqGetInfo()

		pReqGetInfo.URL = types.NewString(getData.URL.String())
		pReqGetInfo.RequestHeaders = types.NewList[datastore_types.DataStoreKeyValue]()
		pReqGetInfo.Size = metaInfo.Size
		pReqGetInfo.RootCACert = types.NewBuffer(getData.RootCACert)
		pReqGetInfo.DataID = metaInfo.DataID

		for key, value := range getData.RequestHeaders {
			header := datastore_types.NewDataStoreKeyValue()
			header.Key = types.NewString(key)
			header.Value = types.NewString(value)

			pReqGetInfo.RequestHeaders = append(pReqGetInfo.RequestHeaders, header)
		}

		pInfos = append(pInfos, pReqGetInfo)
		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pInfos.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetObjectInfos
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
