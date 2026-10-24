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

func (commonProtocol *CommonProtocol) changeMetasV1(err error, packet nex.PacketInterface, callID uint32, dataIDs types.List[types.UInt64], params types.List[datastore_types.DataStoreChangeMetaParamV1], transactional types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	if len(dataIDs) > datastore_constants.BatchProcessingCapacity || len(params) > datastore_constants.BatchProcessingCapacity {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	// * A request is only valid if it's either many DataIDs with one param,
	// * or no DataIDs with many params
	useDataIDList := len(dataIDs) != 0
	if (useDataIDList && len(params) != 1) || (!useDataIDList && len(params) == 0) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	targets := len(params)
	if useDataIDList {
		targets = len(dataIDs)
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
	}

	// TODO - Refactor this, this can make hundreds of database calls in one go
	for i := 0; i < targets; i++ {
		// * Checking the previous object for errors was just a simpler refactor.
		// * The final object error is checked after the loop because lazy
		if bool(transactional) && i > 0 && pResults[i-1].IsError() {
			break
		}

		var dataID types.UInt64
		param := params[0]

		// * V1 has no persistence target
		if useDataIDList {
			dataID = dataIDs[i]
		} else {
			param = params[i]
			dataID = param.DataID
		}

		metaInfo, updatePassword, errCode := database.GetUpdateObjectInfoByDataID(db, dataID)
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

		// * If the object is pending or rejected, only the owner can interact with it
		if metaInfo.OwnerID != connection.PID() && (metaInfo.Status == datastore_constants.DataStatusPending || metaInfo.Status == datastore_constants.DataStatusRejected) {
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
			continue
		}

		errCode = database.UpdateObjectMetadataV1(db, metaInfo, param)
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
	rmcResponse.MethodID = datastore.MethodChangeMetasV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
