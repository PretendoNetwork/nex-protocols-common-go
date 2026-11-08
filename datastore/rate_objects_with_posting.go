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

func (commonProtocol *CommonProtocol) rateObjectsWithPosting(err error, packet nex.PacketInterface, callID uint32, targets types.List[datastore_types.DataStoreRatingTarget], rateParams types.List[datastore_types.DataStoreRateObjectParam], postParams types.List[datastore_types.DataStorePreparePostParam], transactional types.Bool, fetchRatings types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Reusing this constant. Name doesn't make sense but it's the same value so w/e
	if len(targets) > datastore_constants.BatchProcessingCapacityPostObject {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	if len(rateParams) > datastore_constants.BatchProcessingCapacityPostObject {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	if len(postParams) > datastore_constants.BatchProcessingCapacityPostObject {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OverCapacity, "change_error")
	}

	// * Sending 1 param is supported, all targets will get the same value applied to them.
	// * Otherwise, params and targets need to have the same number of elements
	if len(rateParams) != 1 && len(rateParams) != len(targets) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if len(postParams) != 1 && len(postParams) != len(targets) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	pRatings := types.NewList[datastore_types.DataStoreRatingInfo]()
	pResults := types.NewList[types.QResult]()
	invalidRatingInfo := datastore_types.NewDataStoreRatingInfo() // * Quick hack to get a zeroed struct

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

		// * New slots have to be initialized before the existing ones are locked
		errCode := database.InitializeRatingSlotsByDataStoreRatingTargets(db, targets, postParams)
		if errCode != nil {
			return nil, errCode
		}

		errCode = database.LockRatingSlotsByDataStoreRatingTargets(db, targets)
		if errCode != nil {
			return nil, errCode
		}
	}

	// TODO - Optimize this, this can make dozens of database calls for a single RMC request
	for i := 0; i < len(targets); i++ {
		// * Checking the previous target for errors was just a simpler refactor.
		// * The final target error is checked after the loop because lazy
		if bool(transactional) && i > 0 && pResults[i-1].IsError() {
			break
		}

		var rateParam datastore_types.DataStoreRateObjectParam
		if len(rateParams) == 1 {
			rateParam = rateParams[0]
		} else {
			rateParam = rateParams[i]
		}

		var postParam datastore_types.DataStorePreparePostParam
		if len(postParams) == 1 {
			postParam = postParams[0]
		} else {
			postParam = postParams[i]
		}

		target := targets[i]

		if target.DataID == types.UInt64(datastore_constants.InvalidDataID) {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		if target.Slot >= types.UInt8(datastore_constants.NumRatingSlot) {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(db, target.DataID)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, rateParam.AccessPassword)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != datastore_constants.DataStatusNone && metaInfo.OwnerID != connection.PID() {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
			continue
		}

		// * A missing slot is set up first
		errCode = database.InitializeObjectRatingSlotByPreparePostParam(db, target.DataID, target.Slot, postParam)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// TODO - Refactor out this hack. We store decoded flags in Postgres, stop manually parsing them again here
		ratingSettings, errCode := database.GetObjectRatingSlotSettings(db, target.DataID, target.Slot)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The unknown flags 0x01, 0x02 and 0x20 make the slot impossible to rate.
		// * 0x01 crashes on official servers, failing the whole request, while 0x02
		// * and 0x20 are just refused
		// TODO - Move these unknown flags to nex-protocols-go

		if ratingSettings.Flag.HasFlag(0x01) {
			return nil, nex.NewError(nex.ResultCodes.PythonCore.Exception, "change_error")
		}

		if ratingSettings.Flag.HasFlag(0x02) || ratingSettings.Flag.HasFlag(0x20) {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		disableSelfRating := ratingSettings.Flag.HasFlag(datastore_constants.RatingFlagDisableSelfRating)

		useMinimum := ratingSettings.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMin)
		useMaximum := ratingSettings.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMax)

		value := rateParam.RatingValue

		if useMinimum && value < ratingSettings.RangeMin {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		if useMaximum && value > ratingSettings.RangeMax {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		if disableSelfRating && metaInfo.OwnerID == connection.PID() {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.OperationNotAllowed))
			continue
		}

		errCode = database.RateObjectSlot(db, manager, target.DataID, target.Slot, connection.PID(), value, ratingSettings)
		if errCode != nil {
			// * This means the rating was about to go above max int64, which kills the whole request
			if errCode.Equals(nex.ResultCodes.PythonCore.Exception) {
				return nil, errCode
			}

			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		if fetchRatings {
			pRating, errCode := database.GetObjectRatingByDataIDAndSlot(db, target.DataID, target.Slot)
			if errCode != nil {
				pRatings = append(pRatings, invalidRatingInfo)
				pResults = append(pResults, types.NewQResult(errCode.ResultCode))
				continue
			}

			pRatings = append(pRatings, pRating)
		}

		pResults = append(pResults, types.NewQResultSuccess(nex.ResultCodes.DataStore.Unknown))

		// TODO - Create a log here manually, or continue to manually create logs at runtime?
	}

	if transactional {
		// * NOW we check the latest targets error
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

	pRatings.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObjectsWithPosting
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
