package datastore

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) unperpetuateObject(err error, packet nex.PacketInterface, callID uint32, persistenceSlotID types.UInt16, deleteLastObject types.Bool) (*nex.RMCMessage, *nex.Error) {
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

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodUnperpetuateObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
