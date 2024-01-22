package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMetas(err error, packet nex.PacketInterface, callID uint32, dataIDs *types.List[*types.PrimitiveU64], param *datastore_types.DataStoreGetMetaParam) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetObjectInfoByDataID == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// TODO - Verify if param.PersistenceTarget is respected? It wouldn't make sense here but who knows

	pMetaInfo := types.NewList[*datastore_types.DataStoreMetaInfo]()
	pResults := types.NewList[*types.QResult]()

	pMetaInfo.Type = datastore_types.NewDataStoreMetaInfo()
	pResults.Type = types.NewQResult(0)

	// * param has an AccessPassword, but it goes unchecked here.
	// * The password would need to be the same for every object
	// * in the input array, which doesn't make any sense. Assuming
	// * it's unused until proven otherwise

	dataIDs.Each(func(_ int, dataID *types.PrimitiveU64) bool {
		objectInfo, errCode := commonProtocol.GetObjectInfoByDataID(dataID)

		if errCode != 0 {
			objectInfo = datastore_types.NewDataStoreMetaInfo()

			pResults.Append(types.NewQResultError(errCode))
		} else {
			errCode = commonProtocol.VerifyObjectPermission(objectInfo.OwnerID, connection.PID(), objectInfo.Permission)
			if errCode != 0 {
				objectInfo = datastore_types.NewDataStoreMetaInfo()

				pResults.Append(types.NewQResultError(errCode))
			} else {
				pResults.Append(types.NewQResultSuccess(nex.Errors.DataStore.Unknown))
			}

			objectInfo.FilterPropertiesByResultOption(param.ResultOption)
		}

		pMetaInfo.Append(objectInfo)

		return false
	})

	rmcResponseStream := nex.NewByteStreamOut(server)

	pMetaInfo.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMetas
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
