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

func (commonProtocol *CommonProtocol) completeUpdateObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreCompleteUpdateParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if param.DataID == types.UInt64(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	// * Only the user who initiated the update can actually complete it
	preparer, errCode := database.GetObjectUpdatePreparer(manager.Database, param.DataID, param.Version)
	if errCode != nil {
		return nil, errCode
	}

	if preparer != connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
	}

	// * Note: The official servers do not seem to validate this against S3.
	// *       Because of this, we do not either. But it might be something
	// *       to add later if it becomes a problem

	if param.IsSuccess {
		if errCode := database.CompleteObjectUpdate(manager.Database, param.DataID, param.Version); errCode != nil {
			return nil, errCode
		}

		// * Always notify the recipients even if it's not the latest version
		if errCode := database.SendUpdateNotifications(manager.Database, manager, param.DataID, connection.PID()); errCode != nil {
			return nil, errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompleteUpdateObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
