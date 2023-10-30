package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMetas(err error, client *nex.Client, callID uint32, dataIDs []uint64, param *datastore_types.DataStoreGetMetaParam) uint32 {
	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	// TODO - Verify if param.PersistenceTarget is respected? It wouldn't make sense here but who knows

	pMetaInfo := make([]*datastore_types.DataStoreMetaInfo, 0, len(dataIDs))
	pResults := make([]*nex.Result, 0, len(dataIDs))

	// * param has an AccessPassword, but it goes unchecked here.
	// * The password would need to be the same for every object
	// * in the input array, which doesn't make any sense. Assuming
	// * it's unused until proven otherwise

	for i := 0; i < len(dataIDs); i++ {
		objectInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDHandler(dataIDs[i])

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
	rmcResponse.SetSuccess(datastore.MethodGetMetas, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
