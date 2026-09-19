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

	if len(dataIDs) > datastore_constants.BatchProcessingCapacity {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * If anything fails, roll it all back
	transaction, err := manager.BeginTransaction()
	if err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	// * Does nothing once the transaction has been committed successfully
	defer transaction.Rollback()

	// TODO - This runs up to BATCH_PROCESSING_CAPACITY*4 (400) queries. Create some native batch versions of these functions?
	for _, dataID := range dataIDs {
		creationDate, errCode := database.ObjectCreationDate(transaction, dataID)
		if errCode != nil {
			return nil, errCode
		}

		// * If 3 hours pass and the upload was not completed, object
		// * is removed. Simulating this removal by just bailing
		if time.Now().UTC().Sub(creationDate) >= 3*time.Hour {
			return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
		}

		objectOwner, errCode := database.ObjectOwner(transaction, dataID)
		if errCode != nil {
			return nil, errCode
		}

		if objectOwner != connection.PID() {
			return nil, nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}

		objectUploaded, errCode := database.ObjectUploaded(transaction, dataID)
		if errCode != nil {
			return nil, errCode
		}

		if objectUploaded {
			return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
		}

		// * Note: The official servers do not seem to validate this against S3.
		// *       Because of this, we do not either. But it might be something
		// *       to add later if it becomes a problem

		if errCode := database.MarkObjectUploaded(transaction, dataID); errCode != nil {
			return nil, errCode
		}
	}

	if err := transaction.Commit(); err != nil {
		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodCompletePostObjects
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterCompletePostObjects != nil {
		go commonProtocol.OnAfterCompletePostObjects(packet, dataIDs)
	}

	return rmcResponse, nil
}
