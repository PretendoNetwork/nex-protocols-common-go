package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMetasMultipleParam(err error, packet nex.PacketInterface, callID uint32, params []*datastore_types.DataStoreGetMetaParam) uint32 {
	if commonDataStoreProtocol.getObjectInfoByPersistenceTargetWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByPersistenceTargetWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender()

	pMetaInfo := make([]*datastore_types.DataStoreMetaInfo, 0, len(params))
	pResults := make([]*nex.Result, 0, len(params))

	for _, param := range params {
		var objectInfo *datastore_types.DataStoreMetaInfo
		var errCode uint32

		// * Real server ignores PersistenceTarget if DataID is set
		if param.DataID == 0 {
			objectInfo, errCode = commonDataStoreProtocol.getObjectInfoByPersistenceTargetWithPasswordHandler(param.PersistenceTarget, param.AccessPassword)
		} else {
			objectInfo, errCode = commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(param.DataID, param.AccessPassword)
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

	rmcResponseStream.WriteListStructure(pMetaInfo)
	rmcResponseStream.WriteListResult(pResults)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodGetMetasMultipleParam, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
