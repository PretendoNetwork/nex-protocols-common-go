package datastore

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func changeMeta(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreChangeMetaParam) (*nex.RMCMessage, uint32) {
	if commonDataStoreProtocol.GetObjectInfoByDataID == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.UpdateObjectPeriodByDataIDWithPassword == nil {
		common_globals.Logger.Warning("UpdateObjectPeriodByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.UpdateObjectMetaBinaryByDataIDWithPassword == nil {
		common_globals.Logger.Warning("UpdateObjectMetaBinaryByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if commonDataStoreProtocol.UpdateObjectDataTypeByDataIDWithPassword == nil {
		common_globals.Logger.Warning("UpdateObjectDataTypeByDataIDWithPassword not defined")
		return nil, nex.Errors.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.DataStore.Unknown
	}

	client := packet.Sender().(*nex.PRUDPClient)

	metaInfo, errCode := commonDataStoreProtocol.GetObjectInfoByDataID(param.DataID)
	if errCode != 0 {
		return nil, errCode
	}

	// TODO - Is this the right permission?
	errCode = commonDataStoreProtocol.VerifyObjectPermission(metaInfo.OwnerID, client.PID(), metaInfo.DelPermission)
	if errCode != 0 {
		return nil, errCode
	}

	if param.ModifiesFlag&0x08 != 0 {
		errCode = commonDataStoreProtocol.UpdateObjectPeriodByDataIDWithPassword(param.DataID, param.Period, param.UpdatePassword)
		if errCode != 0 {
			return nil, errCode
		}
	}

	if param.ModifiesFlag&0x10 != 0 {
		errCode = commonDataStoreProtocol.UpdateObjectMetaBinaryByDataIDWithPassword(param.DataID, param.MetaBinary, param.UpdatePassword)
		if errCode != 0 {
			return nil, errCode
		}
	}

	if param.ModifiesFlag&0x80 != 0 {
		errCode = commonDataStoreProtocol.UpdateObjectDataTypeByDataIDWithPassword(param.DataID, param.DataType, param.UpdatePassword)
		if errCode != 0 {
			return nil, errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMeta
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
