package datastore

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) completePostObjects(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64]) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	if len(dataIDs) > int(datastore_constants.BatchProcessingCapacity) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// TODO - Add rollback for when error occurs

	for _, dataID := range dataIDs {
		creationDate, errCode := database.ObjectCreationDate(manager, dataID)
		if errCode != nil {
			return nil, errCode
		}

		// * If 3 hours pass and the upload was not completed, object
		// * is removed. Simulating this removal by just bailing
		if time.Now().UTC().Sub(creationDate) >= 3*time.Hour {
			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}

		objectOwner, errCode := database.ObjectOwner(manager, dataID)
		if errCode != nil {
			return nil, errCode
		}

		if objectOwner != connection.PID() {
			return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
		}

		objectEnabled, errCode := database.ObjectEnabled(manager, dataID)
		if errCode != nil {
			return nil, errCode
		}

		if objectEnabled {
			return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
		}
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObjects
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
