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

	// TODO - Add rollback for when error occurs

	if persistenceSlotID >= types.UInt16(datastore_constants.NumPersistenceSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	metaInfo, _, errCode := database.GetAccessObjectInfoByDataID(manager, dataID)
	if errCode != nil {
		return nil, errCode
	}

	// * Only an object owner can perpetuate it
	if metaInfo.OwnerID != connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	// TODO - All of this can probably be done in a single SQL query

	oldDataID, errCode := database.GetPerpetuatedObjectID(manager, connection.PID(), persistenceSlotID)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on persisting object: %s", errCode.Error())
		return nil, errCode
	}

	if oldDataID != datastore_constants.InvalidDataID {
		errCode := database.UnperpetuateObjectByDataID(manager, oldDataID, deleteLastObject)
		if errCode != nil {
			common_globals.Logger.Errorf("Error on unperpetuating object: %s", errCode.Error())
			return nil, errCode
		}
	}

	errCode = database.PerpetuateObject(manager, connection.PID(), persistenceSlotID, uint64(dataID))
	if errCode != nil {
		common_globals.Logger.Errorf("Error on perpetuating object: %s", errCode.Error())
		return nil, errCode
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPerpetuateObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
