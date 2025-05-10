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

	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	pMetaInfo := types.NewList[datastore_types.DataStoreMetaInfo]()
	pResults := types.NewList[types.QResult]()
	invalidMetaInfo := datastore_types.NewDataStoreMetaInfo() // * Quick hack to get a zeroed struct

	// * param.DataID and param.PersistenceTarget are ignored here
	for _, dataID := range dataIDs {
		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, dataID)
		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * NOTE: IF THE PASSWORD IS SET, THIS WILL NOT WORK FOR MULTIPLE OBJECTS!
		// * THIS SHOULD REALLY ONLY BE CALLED WITHOUT A PASSWORD SET
		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		metaInfo, errCode = database.GetObjectMetaInfoByDataIDWithResultOption(manager, dataID, param.ResultOption)
		if errCode != nil {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			pMetaInfo = append(pMetaInfo, invalidMetaInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		pMetaInfo = append(pMetaInfo, metaInfo)
		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
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
