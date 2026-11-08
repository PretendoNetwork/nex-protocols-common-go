package datastore

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) deleteObjects(err error, packet nex.PacketInterface, callID uint32, params types.List[datastore_types.DataStoreDeleteParam], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if len(params) > datastore_constants.BatchProcessingCapacity {
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

		errCode := database.LockObjectsByDataStoreDeleteParams(db, params)
		if errCode != nil {
			return nil, errCode
		}
	}

	// TODO - Refactor this, this can make up to 200 database queries and 100 gRPC calls in one go
	for i, param := range params {
		// * Checking the previous param for errors was just a simpler refactor.
		// * The final param error is checked after the loop because lazy
		if bool(transactional) && i > 0 && pResults[i-1].IsError() {
			break
		}

		metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(db, param.DataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// TODO - Move this to VerifyObjectUpdatePermission?
		// * Objects in the DataID range 900,000-999,999 belong to the system, and a
		// * normal client is not allowed to touch them
		// * https://nintendo.wiki/wiki/Nintendo_Network/NEX/Protocols/DataStoreProtocol#Object_Data_ID
		if metaInfo.DataID < 1000000 {
			// * Unsure if this is the correct error, but it feels right
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.OperationNotAllowed))
			continue
		}

		errCode = manager.VerifyObjectUpdatePermission(connection.PID(), metaInfo, updatePassword, param.UpdatePassword)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = database.DeleteObject(db, param.DataID)
		if errCode != nil {
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))
	}

	if transactional {
		// * NOW we check the latest params error
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
	rmcResponse.MethodID = datastore.MethodDeleteObjects
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
