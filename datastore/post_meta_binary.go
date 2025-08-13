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

func (commonProtocol *CommonProtocol) postMetaBinary(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStorePreparePostParam) (*nex.RMCMessage, *nex.Error) {
	// * This method functions identically to DataStore::PreparePostObject, except
	// * these objects do not use the file server. This is a more light-weight,
	// * and faster, way to manage "files" that are sufficently small

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if param.Size != 0 {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * Server sets this if not set already
	notUseFileServer := (param.Flag & types.UInt32(datastore_constants.DataFlagNotUseFileServer)) != 0
	if !notUseFileServer {
		param.Flag |= types.UInt32(datastore_constants.DataFlagNotUseFileServer)
	}

	dataID, errCode := database.InsertObjectByPreparePostParam(manager, connection.PID(), param)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on object init: %s", errCode.Error())
		return nil, errCode
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	errCode = database.InsertObjectRatingSettings(manager, dataID, param.RatingInitParams)
	if errCode != nil {
		common_globals.Logger.Errorf("Error on rating init: %s", errCode.Error())
		return nil, errCode
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	if param.PersistenceInitParam.PersistenceSlotID != types.UInt16(datastore_constants.InvalidPersistenceSlotID) {
		slot := param.PersistenceInitParam.PersistenceSlotID
		oldDataID, err := database.GetPerpetuatedObjectID(manager, connection.PID(), slot)
		if err != nil {
			common_globals.Logger.Errorf("Error on persisting object: %s", err.Error())
			return nil, err
		}

		if oldDataID != datastore_constants.InvalidDataID {
			err := database.UnperpetuateObjectByDataID(manager, oldDataID, param.PersistenceInitParam.DeleteLastObject)
			if err != nil {
				common_globals.Logger.Errorf("Error on unperpetuating object: %s", err.Error())
				return nil, err
			}
		}

		err = database.PerpetuateObject(manager, connection.PID(), param.PersistenceInitParam.PersistenceSlotID, dataID)
		if err != nil {
			common_globals.Logger.Errorf("Error on perpetuating object: %s", err.Error())
			return nil, err
		}
	}

	// TODO - Should this be moved inside InsertObjectByPreparePostParam?
	notifyAccessRecipientsOnCreation := (param.Flag & types.UInt32(datastore_constants.DataFlagUseNotificationOnPost)) != 0
	if notifyAccessRecipientsOnCreation {
		recipientIDs, errCode := manager.GetNotificationRecipients(connection.PID(), param.Permission)
		if errCode != nil {
			common_globals.Logger.Errorf("Error on getting notification recipients: %s", errCode.Error())
			return nil, errCode
		}

		for _, recipientID := range recipientIDs {
			errCode = database.SendNotification(manager, dataID, recipientID, connection.PID())
			if errCode != nil {
				common_globals.Logger.Errorf("Error on sending notification: %s", errCode.Error())
				return nil, errCode
			}
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	rmcResponseStream.WriteUInt64LE(dataID)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodPostMetaBinary
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterPostMetaBinary != nil {
		go commonProtocol.OnAfterPostMetaBinary(packet, param)
	}

	return rmcResponse, nil
}
