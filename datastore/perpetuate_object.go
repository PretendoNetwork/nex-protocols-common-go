package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) perpetuateObject(err error, packet nex.PacketInterface, callID uint32, persistenceSlotID types.UInt16, dataID types.UInt64, deleteLastObject types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Defined up here unlike the other RMC handlers because theres
	// * so many early bails here. I dont super love this pattern, try
	// * to find a way around it
	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPerpetuateObject
	rmcResponse.CallID = callID

	// TODO - Add rollback for when error occurs

	if persistenceSlotID >= types.UInt16(datastore_constants.NumPersistenceSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * Trying to use InvalidDataID on an empty slot does nothing, but
	// * a slot with an object already in it does error for some reason
	if dataID == types.UInt64(datastore_constants.InvalidDataID) {
		oldDataID, errCode := database.GetPerpetuatedObjectID(manager.Database, connection.PID(), persistenceSlotID)
		if errCode != nil {
			return nil, errCode
		}

		if oldDataID != datastore_constants.InvalidDataID {
			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}

		return rmcResponse, nil
	}

	metaInfo, _, errCode := database.GetAccessObjectInfoByDataID(manager.Database, dataID)
	if errCode != nil {
		return nil, errCode
	}

	// * Only an object owner can perpetuate it
	if metaInfo.OwnerID != connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
	}

	// TODO - All of this can probably be done in a single SQL query

	oldDataID, errCode := database.GetPerpetuatedObjectID(manager.Database, connection.PID(), persistenceSlotID)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on persisting object: %s", errCode.Error())
		return nil, errCode
	}

	currentSlotID, errCode := database.GetPersistenceSlotIDByDataID(manager.Database, dataID)
	if errCode != nil {
		return nil, errCode
	}

	// * Trying to move an existing persisted object into an empty slot does nothing
	if currentSlotID != datastore_constants.InvalidPersistenceSlotID && currentSlotID != uint16(persistenceSlotID) && oldDataID == datastore_constants.InvalidDataID {
		return rmcResponse, nil
	}

	// * Trying to move an existing persisted object into an occupied slot is not allowed
	if currentSlotID != datastore_constants.InvalidPersistenceSlotID && currentSlotID != uint16(persistenceSlotID) && oldDataID != datastore_constants.InvalidDataID {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	// * If the same object is being put into the same slot, it has no "last object"
	// * to delete
	if oldDataID != datastore_constants.InvalidDataID && oldDataID != uint64(dataID) {
		errCode := database.UnperpetuateObjectByDataID(manager.Database, oldDataID, deleteLastObject)
		if errCode != nil {
			common_globals.Logger.Errorf("Error on unperpetuating object: %s", errCode.Error())
			return nil, errCode
		}
	}

	errCode = database.PerpetuateObject(manager.Database, connection.PID(), persistenceSlotID, uint64(dataID))
	if errCode != nil {
		common_globals.Logger.Errorf("Error on perpetuating object: %s", errCode.Error())
		return nil, errCode
	}

	return rmcResponse, nil
}
