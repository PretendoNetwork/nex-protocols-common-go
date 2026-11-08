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

func (commonProtocol *CommonProtocol) changeMeta(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreChangeMetaParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if common_globals.HasSurrogatePair(param.Name) || common_globals.HasSurrogatePair(param.Tags...) {
		return nil, nex.NewError(nex.ResultCodes.PythonCore.ConversionError, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	var metaInfo datastore_types.DataStoreMetaInfo
	var updatePassword types.UInt64
	var errCode *nex.Error

	// * The persistence target is used whenever it has a slot, even if it has no ownerId
	if param.PersistenceTarget.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByPersistenceTarget(manager.Database, param.PersistenceTarget)
	} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
		metaInfo, updatePassword, errCode = database.GetUpdateObjectInfoByDataID(manager.Database, param.DataID)
	} else {
		// * If both the PersistenceTarget and DataID are not set, bail
		errCode = nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	if errCode != nil {
		return nil, errCode
	}

	// TODO - Move this to VerifyObjectUpdatePermission?
	// * Objects in the DataID range 900,000-999,999 belong to the system, and a
	// * normal client is not allowed to touch them
	// * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
	if metaInfo.DataID < 1000000 {
		// * Unsure if this is the correct error, but it feels right
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
	if errCode != nil {
		return nil, errCode
	}

	// * If the object is pending or rejected, only the owner can interact with it
	if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == datastore_constants.DataStatusPending || metaInfo.Status == datastore_constants.DataStatusRejected) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
	}

	errCode = database.UpdateObjectMetadata(manager.Database, manager, metaInfo, param)
	if errCode != nil {
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodChangeMeta
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterChangeMeta != nil {
		go commonProtocol.OnAfterChangeMeta(packet, param)
	}

	return rmcResponse, nil
}
