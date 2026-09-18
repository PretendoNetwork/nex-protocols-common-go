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

	// * The persistence target is used whenever it has a slot, even if it has no ownerId
	if param.PersistenceTarget.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		metaInfo, accessPassword, errCode = database.GetAccessObjectInfoByPersistenceTarget(manager, param.PersistenceTarget)
	} else if param.DataID != types.UInt64(datastore_constants.InvalidDataID) {
		metaInfo, accessPassword, errCode = database.GetAccessObjectInfoByDataID(manager, param.DataID)
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

	pMetaInfo, errCode := database.GetObjectMetaInfoByDataIDWithResultOption(manager, metaInfo.DataID, param.ResultOption)
	if errCode != nil {
		return nil, errCode
	}

	// TODO - Move this to VerifyObjectAccessPermission?
	if pMetaInfo.Status != datastore_constants.DataStatusNone {
		// * Rejected objects behave as if they do not exist to everyone, including the owner.
		// * The only exception is SearchObject/SearchObjectLight, using either
		// * SEARCH_TYPE_OWN_REJECTED or SEARCH_TYPE_OWN_ALL
		if pMetaInfo.Status == datastore_constants.DataStatusRejected {
			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}

		// * Objects under review can still be viewed by their owner, but not by normal users
		if pMetaInfo.OwnerID != connection.PID() {
			if pMetaInfo.Status == datastore_constants.DataStatusPending {
				return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
			}

			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}
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
