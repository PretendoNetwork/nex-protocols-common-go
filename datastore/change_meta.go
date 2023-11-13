package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func changeMeta(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreChangeMetaParam) uint32 {
	if commonDataStoreProtocol.getObjectInfoByDataIDHandler == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.updateObjectPeriodByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("UpdateObjectPeriodByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.updateObjectMetaBinaryByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("UpdateObjectMetaBinaryByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.updateObjectDataTypeByDataIDWithPasswordHandler == nil {
		common_globals.Logger.Warning("UpdateObjectDataTypeByDataIDWithPassword not defined")
		return nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	metaInfo, errCode := commonDataStoreProtocol.getObjectInfoByDataIDHandler(param.DataID)
	if errCode != 0 {
		return errCode
	}

	// TODO - Is this the right permission?
	errCode = commonDataStoreProtocol.VerifyObjectPermission(metaInfo.OwnerID, client.PID().LegacyValue(), metaInfo.DelPermission)
	if errCode != 0 {
		return errCode
	}

	if param.ModifiesFlag&0x08 != 0 {
		errCode = commonDataStoreProtocol.updateObjectPeriodByDataIDWithPasswordHandler(param.DataID, param.Period, param.UpdatePassword)
		if errCode != 0 {
			return errCode
		}
	}

	if param.ModifiesFlag&0x10 != 0 {
		errCode = commonDataStoreProtocol.updateObjectMetaBinaryByDataIDWithPasswordHandler(param.DataID, param.MetaBinary, param.UpdatePassword)
		if errCode != 0 {
			return errCode
		}
	}

	if param.ModifiesFlag&0x80 != 0 {
		errCode = commonDataStoreProtocol.updateObjectDataTypeByDataIDWithPasswordHandler(param.DataID, param.DataType, param.UpdatePassword)
		if errCode != 0 {
			return errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMeta
	rmcResponse.CallID = callID

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PRUDPPacketInterface

	if commonDataStoreProtocol.server.PRUDPVersion == 0 {
		responsePacket, _ = nex.NewPRUDPPacketV0(client, nil)
	} else {
		responsePacket, _ = nex.NewPRUDPPacketV1(client, nil)
	}

	responsePacket.SetType(nex.DataPacket)
	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)
	responsePacket.SetSourceStreamType(packet.(nex.PRUDPPacketInterface).DestinationStreamType())
	responsePacket.SetSourcePort(packet.(nex.PRUDPPacketInterface).DestinationPort())
	responsePacket.SetDestinationStreamType(packet.(nex.PRUDPPacketInterface).SourceStreamType())
	responsePacket.SetDestinationPort(packet.(nex.PRUDPPacketInterface).SourcePort())
	responsePacket.SetPayload(rmcResponseBytes)

	commonDataStoreProtocol.server.Send(responsePacket)

	return 0
}
