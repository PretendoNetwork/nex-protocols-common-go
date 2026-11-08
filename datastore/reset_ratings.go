package datastore

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
)

func (commonProtocol *CommonProtocol) resetRatings(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * This IS BatchProcessingCapacity, unlike other rating methods
	if len(dataIDs) > datastore_constants.BatchProcessingCapacity {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	pResults := types.NewList[types.QResult]()

	var db common_globals.Database = manager.Database
	var transaction *sql.Tx

	if transactional {
		var err error
		transaction, err = manager.BeginTransaction()
		if err != nil {
			// TODO - Send more specific errors?
			return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		// * Does nothing once the transaction has been committed successfully
		defer transaction.Rollback()

		db = transaction

		errCode := database.LockRatingSlotsByDataIDs(db, dataIDs)
		if errCode != nil {
			return nil, errCode
		}
	}

	for i, dataID := range dataIDs {
		// * Checking the previous object for errors was just a simpler refactor.
		// * The final object error is checked after the loop because lazy
		if bool(transactional) && i > 0 && pResults[i-1].IsError() {
			break
		}

		if dataID == types.UInt64(datastore_constants.InvalidDataID) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		metaInfo, objectUpdatePassword, errCode := database.GetUpdateObjectInfoByDataID(db, dataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, objectUpdatePassword, 0)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * Unlike most methods, not even the owner can interact with objects in this case
		if metaInfo.Status != datastore_constants.DataStatusNone {
			if metaInfo.OwnerID == connection.PID() {
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			} else {
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
			}

			continue
		}

		errCode = database.DeleteObjectRatings(db, dataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	if transactional {
		// * NOW we check the latest objects error
		if len(pResults) != 0 && pResults[len(pResults)-1].IsError() {
			return nil, nex.NewError(uint32(pResults[len(pResults)-1]), "change_error")
		}

		if err := transaction.Commit(); err != nil {
			// TODO - Send more specific errors?
			return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
		}

		// * Results list is empty when transactions are used
		pResults = types.NewList[types.QResult]()
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodResetRatings
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
