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

func (commonProtocol *CommonProtocol) getPersistenceInfo(err error, packet nex.PacketInterface, callID uint32, ownerID types.PID, persistenceSlotID types.UInt16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Why Nintendo didn't just use this directly, we may never know
	persistenceTarget := datastore_types.NewDataStorePersistenceTarget()
	persistenceTarget.OwnerID = ownerID
	persistenceTarget.PersistenceSlotID = persistenceSlotID

	metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByPersistenceTarget(manager, persistenceTarget)
	if errCode != nil {
		return nil, errCode
	}

	// * Anyone with access permissions can view this, but they don't allow a password to be sent
	errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, 0)
	if errCode != nil {
		return nil, errCode
	}

	// * The owner of an object can always view their objects, but normal users cannot
	if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
		if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
		}

		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	pPersistenceInfo := datastore_types.NewDataStorePersistenceInfo()
	pPersistenceInfo.OwnerID = ownerID
	pPersistenceInfo.PersistenceSlotID = persistenceSlotID
	pPersistenceInfo.DataID = metaInfo.DataID

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pPersistenceInfo.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodGetPersistenceInfo
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
