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

func (commonProtocol *CommonProtocol) getPersistenceInfos(err error, packet nex.PacketInterface, callID uint32, ownerID types.PID, persistenceSlotIDs types.List[types.UInt16]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Every user can only have 16 objects
	if len(persistenceSlotIDs) > int(datastore_constants.NumPersistenceSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	pPersistenceInfo := types.NewList[datastore_types.DataStorePersistenceInfo]()
	pResults := types.NewList[types.QResult]()
	invalidPersistenceInfo := datastore_types.NewDataStorePersistenceInfo() // * Quick hack to get a zeroed struct

	for _, persistenceSlotID := range persistenceSlotIDs {
		// * Why Nintendo didn't just use this directly, we may never know
		persistenceTarget := datastore_types.NewDataStorePersistenceTarget()
		persistenceTarget.OwnerID = ownerID
		persistenceTarget.PersistenceSlotID = persistenceSlotID

		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByPersistenceTarget(manager, persistenceTarget)
		if errCode != nil {
			pPersistenceInfo = append(pPersistenceInfo, invalidPersistenceInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * Anyone with access permissions can view this, but they don't allow a password to be sent
		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, 0)
		if errCode != nil {
			pPersistenceInfo = append(pPersistenceInfo, invalidPersistenceInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			pPersistenceInfo = append(pPersistenceInfo, invalidPersistenceInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		persistenceInfo := datastore_types.NewDataStorePersistenceInfo()
		persistenceInfo.OwnerID = ownerID
		persistenceInfo.PersistenceSlotID = persistenceSlotID
		persistenceInfo.DataID = metaInfo.DataID

		pPersistenceInfo = append(pPersistenceInfo, persistenceInfo)
		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.UnderReviewing))
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pPersistenceInfo.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetPersistenceInfos
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
