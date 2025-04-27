package datastore

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) completePostObject(err error, packet nex.PacketInterface, callID uint32, param datastore_types.DataStoreCompletePostParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	creationDate, errCode := database.ObjectCreationDate(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	// * If 3 hours pass and the upload was not completed, object
	// * is removed. Simulating this removal by just bailing
	if time.Now().UTC().Sub(creationDate) >= 3*time.Hour {
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	objectOwner, errCode := database.ObjectOwner(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	if objectOwner != connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	objectEnabled, errCode := database.ObjectEnabled(manager, param.DataID)
	if errCode != nil {
		return nil, errCode
	}

	if objectEnabled {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	// * Note: The official servers do not seem to validate this against S3.
	// *       Because of this, we do not either. But it might be something
	// *       to add later if it becomes a problem

	if param.IsSuccess {
		if errCode := database.EnableObject(manager, param.DataID); errCode != nil {
			return nil, errCode
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObject
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCompletePostObject != nil {
		go commonProtocol.OnAfterCompletePostObject(packet, param)
	}

	return rmcResponse, nil
}
