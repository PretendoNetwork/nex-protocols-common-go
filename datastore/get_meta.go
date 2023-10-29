package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func getMeta(err error, client *nex.Client, callID uint32, param *datastore_types.DataStoreGetMetaParam) uint32 {
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

	var pMetaInfo *datastore_types.DataStoreMetaInfo
	var errCode uint32

	// * Real server ignores PersistenceTarget if DataID is set
	if param.DataID == 0 {
		pMetaInfo, errCode = commonDataStoreProtocol.getObjectInfoByPersistenceTargetWithPasswordHandler(param.PersistenceTarget, param.AccessPassword)
	} else {
		pMetaInfo, errCode = commonDataStoreProtocol.getObjectInfoByDataIDWithPasswordHandler(param.DataID, param.AccessPassword)
	}

	if errCode != 0 {
		return errCode
	}

	errCode = commonDataStoreProtocol.VerifyObjectPermission(pMetaInfo.OwnerID, client.PID(), pMetaInfo.Permission)
	if errCode != 0 {
		return errCode
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
		pMetaInfo.Tags = make([]string, 0)
	}

	if param.ResultOption&0x2 == 0 {
		pMetaInfo.Ratings = make([]*datastore_types.DataStoreRatingInfoWithSlot, 0)
	}

	if param.ResultOption&0x4 == 0 {
		pMetaInfo.MetaBinary = make([]byte, 0)
	}

	rmcResponseStream := nex.NewStreamOut(commonDataStoreProtocol.server)
	rmcResponseStream.WriteStructure(pMetaInfo)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(datastore.ProtocolID, callID)
	rmcResponse.SetSuccess(datastore.MethodGetMeta, rmcResponseBody)

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
