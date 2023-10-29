package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMetasMultipleParam(err error, client *nex.Client, callID uint32, params []*datastore_types.DataStoreGetMetaParam) uint32 {
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
			// TODO - Maybe this should be broken out into a util function in globals?
			objectInfo = datastore_types.NewDataStoreMetaInfo()
			objectInfo.DataID = 0
			objectInfo.OwnerID = 0
			objectInfo.Size = 0
			objectInfo.Name = ""
			objectInfo.DataType = 0
			objectInfo.MetaBinary = []byte{}
			objectInfo.Permission = datastore_types.NewDataStorePermission()
			objectInfo.Permission.Permission = 0
			objectInfo.Permission.RecipientIDs = []uint32{}
			objectInfo.DelPermission = datastore_types.NewDataStorePermission()
			objectInfo.DelPermission.Permission = 0
			objectInfo.DelPermission.RecipientIDs = []uint32{}
			objectInfo.CreatedTime = nex.NewDateTime(0)
			objectInfo.UpdatedTime = nex.NewDateTime(0)
			objectInfo.Period = 0
			objectInfo.Status = 0
			objectInfo.ReferredCnt = 0
			objectInfo.ReferDataID = 0
			objectInfo.Flag = 0
			objectInfo.ReferredTime = nex.NewDateTime(0)
			objectInfo.ExpireTime = nex.NewDateTime(0)
			objectInfo.Tags = []string{}
			objectInfo.Ratings = []*datastore_types.DataStoreRatingInfoWithSlot{}

			pResults = append(pResults, nex.NewResultError(errCode))
		} else {
			errCode = commonDataStoreProtocol.VerifyObjectPermission(objectInfo.OwnerID, client.PID(), objectInfo.Permission)
			if errCode != 0 {
				// TODO - Maybe this should be broken out into a util function in globals?
				objectInfo = datastore_types.NewDataStoreMetaInfo()
				objectInfo.DataID = 0
				objectInfo.OwnerID = 0
				objectInfo.Size = 0
				objectInfo.Name = ""
				objectInfo.DataType = 0
				objectInfo.MetaBinary = []byte{}
				objectInfo.Permission = datastore_types.NewDataStorePermission()
				objectInfo.Permission.Permission = 0
				objectInfo.Permission.RecipientIDs = []uint32{}
				objectInfo.DelPermission = datastore_types.NewDataStorePermission()
				objectInfo.DelPermission.Permission = 0
				objectInfo.DelPermission.RecipientIDs = []uint32{}
				objectInfo.CreatedTime = nex.NewDateTime(0)
				objectInfo.UpdatedTime = nex.NewDateTime(0)
				objectInfo.Period = 0
				objectInfo.Status = 0
				objectInfo.ReferredCnt = 0
				objectInfo.ReferDataID = 0
				objectInfo.Flag = 0
				objectInfo.ReferredTime = nex.NewDateTime(0)
				objectInfo.ExpireTime = nex.NewDateTime(0)
				objectInfo.Tags = []string{}
				objectInfo.Ratings = []*datastore_types.DataStoreRatingInfoWithSlot{}

				pResults = append(pResults, nex.NewResultError(errCode))
			} else {
				pResults = append(pResults, nex.NewResultSuccess(nex.Errors.DataStore.Unknown))
			}

			// * This is kind of backwards.
			// * The database pulls this data
			// * by default, so it can be done
			// * in a single query. So instead
			// * of checking if a flag *IS*
			// * set, and conditionally *ADDING*
			// * the fields, we check if a flag
			// * is *NOT* set and conditionally
			// * *REMOVE* the field
			if param.ResultOption&0x1 == 0 {
				objectInfo.Tags = make([]string, 0)
			}

			if param.ResultOption&0x2 == 0 {
				objectInfo.Ratings = make([]*datastore_types.DataStoreRatingInfoWithSlot, 0)
			}

			if param.ResultOption&0x4 == 0 {
				objectInfo.MetaBinary = make([]byte, 0)
			}
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

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
