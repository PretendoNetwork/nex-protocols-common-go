package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMetasMultipleParam(err error, packet nex.PacketInterface, callID uint32, params []*datastore_types.DataStoreGetMetaParam) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.GetObjectInfoByPersistenceTargetWithPassword == nil {
		common_globals.Logger.Warning("GetObjectInfoByPersistenceTargetWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.GetObjectInfoByDataIDWithPassword == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	pMetaInfo := make([]*datastore_types.DataStoreMetaInfo, 0, len(params))
	pResults := make([]*nex.Result, 0, len(params))

	for _, param := range params {
		var objectInfo *datastore_types.DataStoreMetaInfo
		var errCode uint32

		// * Real server ignores PersistenceTarget if DataID is set
		if param.DataID == 0 {
			objectInfo, errCode = commonDataStoreProtocol.GetObjectInfoByPersistenceTargetWithPassword(param.PersistenceTarget, param.AccessPassword)
		} else {
			objectInfo, errCode = commonDataStoreProtocol.GetObjectInfoByDataIDWithPassword(param.DataID, param.AccessPassword)
		}

		if errCode != 0 {
			objectInfo = datastore_types.NewDataStoreMetaInfo()

			pResults = append(pResults, nex.NewResultError(errCode))
		} else {
			errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
			if errCode != 0 {
				objectInfo = datastore_types.NewDataStoreMetaInfo()

				pResults = append(pResults, nex.NewResultError(errCode))
			} else {
				pResults = append(pResults, nex.NewResultSuccess(nex.Errors.DataStore.Unknown))
			}

			objectInfo.FilterPropertiesByResultOption(param.ResultOption)
		}

		pMetaInfo = append(pMetaInfo, objectInfo)
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)

	nex.StreamWriteListStructure(rmcResponseStream, pMetaInfo)
	rmcResponseStream.WriteListResult(pResults)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMetasMultipleParam
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
