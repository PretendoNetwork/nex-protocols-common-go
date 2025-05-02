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

func (commonProtocol *CommonProtocol) getMetasMultipleParam(err error, packet nex.PacketInterface, callID uint32, params types.List[datastore_types.DataStoreGetMetaParam]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if len(params) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pMetaInfo := types.NewList[datastore_types.DataStoreMetaInfo]()
	pResults := types.NewList[types.QResult]()
	invalidMetaInfo := datastore_types.NewDataStoreMetaInfo()                 // * Quick hack to get a zeroed struct
	invalidResult := types.NewQResultError(nex.ResultCodes.DataStore.Unknown) // TODO - Temp. Customize this per error

	// * param.DataID and param.PersistenceTarget are ignored here
	for _, param := range params {
		var metaInfo datastore_types.DataStoreMetaInfo
		var accessPassword types.UInt64
		var errCode *nex.Error

		if param.PersistenceTarget.OwnerID != 0 {
			metaInfo, accessPassword, errCode = database.GetAccessObjectInfoByPersistenceTarget(manager, param.PersistenceTarget)
		} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
			metaInfo, accessPassword, errCode = database.GetAccessObjectInfoByDataID(manager, param.DataID)
		} else {
			// * If both the PersistenceTarget and DataID are not set, bail
			errCode = nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
		}

		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo.Copy().(datastore_types.DataStoreMetaInfo))
			pResults = append(pResults, invalidResult.Copy().(types.QResult))
			continue
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo.Copy().(datastore_types.DataStoreMetaInfo))
			pResults = append(pResults, invalidResult.Copy().(types.QResult))
			continue
		}

		metaInfo, errCode = database.GetObjectMetaInfoByDataIDWithResultOption(manager, metaInfo.DataID, param.ResultOption)
		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo.Copy().(datastore_types.DataStoreMetaInfo))
			pResults = append(pResults, invalidResult.Copy().(types.QResult))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo.Copy().(datastore_types.DataStoreMetaInfo))
			pResults = append(pResults, invalidResult.Copy().(types.QResult))
			continue
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfo.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMetasMultipleParam
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
