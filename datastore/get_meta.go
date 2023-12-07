package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMeta(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreGetMetaParam) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.getObjectInfoByPersistenceTargetWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByPersistenceTargetWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	var pMetaInfo *datastore_types.DataStoreMetaInfo
	var errCode uint32

	// * Real server ignores PersistenceTarget if DataID is set
	if param.DataID == 0 {
		pMetaInfo, errCode = commonDataStoreProtocol.getObjectInfoByPersistenceTargetWithPasswordHandler(param.PersistenceTarget, param.AccessPassword)
	} else {
		pMetaInfo, errCode = commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(param.DataID, param.AccessPassword)
	}

	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(pMetaInfo.OwnerID, client.PID(), pMetaInfo.Permission)
	if errCode != 0 {
		return nil, errCode
	}

	pMetaInfo.FilterPropertiesByResultOption(param.ResultOption)

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)
	rmcResponseStream.WriteStructure(pMetaInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMeta
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
