package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) getMeta(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreGetMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	var metaInfo datastore_types.DataStoreMetaInfo
	var accessPassword types.UInt64
	var errCode *nex.Error

	if param.PersistenceTarget.OwnerID != 0 {
		metaInfo, accessPassword, errCode = database.GetGetMetaObjectInfoByPersistenceTarget(manager, param.PersistenceTarget)
	} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
		metaInfo, accessPassword, errCode = database.GetGetMetaObjectInfoByDataID(manager, param.DataID)
	} else {
		// * If both the PersistenceTarget and DataID are not set, bail
		errCode = nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
	if errCode != nil {
		return nil, errCode
	}

	pMetaInfo, errCode := database.GetObjectMetaInfoByDataIDWithResultOption(manager, param.DataID, param.ResultOption)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pMetaInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetMeta
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterGetMeta != nil {
		go commonProtocol.OnAfterGetMeta(packet, param)
	}

	return rmcResponse, nil
}
