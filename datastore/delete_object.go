package datastore

import (
	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

func deleteObject(err error, packet nex.PacketInterface, callID uint32, param *datastore_types.DataStoreDeleteParam) (*nex.RMCMessage, uint32) {
	if commonProtocol.GetObjectInfoByDataID == nil {
		common_globals.Logger.Warning("GetObjectInfoByDataID not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if commonProtocol.DeleteObjectByDataIDWithPassword == nil {
		common_globals.Logger.Warning("DeleteObjectByDataIDWithPassword not defined")
		return nil, nex.ResultCodes.Core.NotImplemented
	}

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.DataStore.Unknown
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	metaInfo, errCode := commonProtocol.GetObjectInfoByDataID(param.DataID)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonProtocol.VerifyObjectPermission(metaInfo.OwnerID, connection.PID(), metaInfo.DelPermission)
	if errCode != 0 {
		return nil, errCode
	}

	errCode = commonProtocol.DeleteObjectByDataIDWithPassword(param.DataID, param.UpdatePassword)
	if errCode != 0 {
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodDeleteObject
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
