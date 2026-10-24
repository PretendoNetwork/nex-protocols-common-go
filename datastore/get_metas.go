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

func (commonProtocol *CommonProtocol) getMetas(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], param datastore_types.DataStoreGetMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if len(dataIDs) > datastore_constants.BatchProcessingCapacity {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pMetaInfo := types.NewList[datastore_types.DataStoreMetaInfo]()
	pResults := types.NewList[types.QResult]()
	invalidMetaInfo := datastore_types.NewDataStoreMetaInfo() // * Quick hack to get a zeroed struct

	// TODO - I really hate this pattern, defining functions this way inside a function SUCKS. Change
	getEntry := func(metaInfo datastore_types.DataStoreMetaInfo, accessPassword types.UInt64, errCode *nex.Error) (datastore_types.DataStoreMetaInfo, types.QResult) {
		if errCode != nil {
			return invalidMetaInfo, types.NewQResult(errCode.ResultCode)
		}

		// * NOTE: IF THE PASSWORD IS SET, THIS WILL NOT WORK FOR MULTIPLE OBJECTS!
		// * THIS SHOULD REALLY ONLY BE CALLED WITHOUT A PASSWORD SET
		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
		if errCode != nil {
			return invalidMetaInfo, types.NewQResult(errCode.ResultCode)
		}

		metaInfo, errCode = database.GetObjectMetaInfoByDataIDWithResultOption(manager.Database, metaInfo.DataID, param.ResultOption)
		if errCode != nil {
			return invalidMetaInfo, types.NewQResult(errCode.ResultCode)
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != datastore_constants.DataStatusNone && metaInfo.OwnerID != connection.PID() {
			return invalidMetaInfo, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing)
		}

		return metaInfo, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown)
	}

	// * If param.PersistenceTarget is used then it's shared for all objects. This saves
	// * up to 300 lookups by just doing it once in that case
	if param.PersistenceTarget.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		metaInfo, result := getEntry(database.GetAccessObjectInfoByPersistenceTarget(manager.Database, param.PersistenceTarget))

		for range dataIDs {
			pMetaInfo = append(pMetaInfo, metaInfo)
			pResults = append(pResults, result)
		}
	} else {
		for _, dataID := range dataIDs {
			metaInfo, result := getEntry(database.GetAccessObjectInfoByDataID(manager.Database, dataID))

			pMetaInfo = append(pMetaInfo, metaInfo)
			pResults = append(pResults, result)
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfo.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMetas
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
