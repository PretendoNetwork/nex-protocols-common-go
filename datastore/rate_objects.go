package datastore

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/datastore/database"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/v2/datastore"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func (commonProtocol *CommonProtocol) rateObjects(err error, packet nex.PacketInterface, callID uint32, targets types.List[datastore_types.DataStoreRatingTarget], params types.List[datastore_types.DataStoreRateObjectParam], transactional types.Bool, fetchRatings types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// * Reusing this constant. Name doesn't make sense but it's the same value so w/e
	if len(targets) > int(datastore_constants.BatchProcessingCapacityPostObject) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if len(params) > int(datastore_constants.BatchProcessingCapacityPostObject) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * Sending 1 param is supported, all targets will get the same value applied to them.
	// * Otherwise, params and targets need to have the same number of elements
	if len(params) != 1 && len(params) != len(targets) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// TODO - Add rollback for when error occurs

	pRatings := types.NewList[datastore_types.DataStoreRatingInfo]()
	pResults := types.NewList[types.QResult]()
	invalidRatingInfo := datastore_types.NewDataStoreRatingInfo() // * Quick hack to get a zeroed struct

	// TODO - Optimize this, this can make dozens of database calls for a single RMC request
	for i := 0; i < len(targets); i++ {
		var param datastore_types.DataStoreRateObjectParam
		if len(params) == 1 {
			param = params[0]
		} else {
			param = params[i]
		}

		target := targets[i]

		if target.DataID == types.UInt64(datastore_constants.InvalidDataID) {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		if target.Slot >= types.UInt8(datastore_constants.NumRatingSlot) {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.InvalidArgument))
			continue
		}

		metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, target.DataID)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		// * The owner of an object can always view their objects, but normal users cannot
		if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
			if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
				pRatings = append(pRatings, invalidRatingInfo)
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.UnderReviewing))
				continue
			}

			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.NotFound))
			continue
		}

		// TODO - Refactor out this hack. We store decoded flags in Postgres, stop manually parsing them again here
		ratingSettings, errCode := database.GetObjectRatingSlotSettings(manager, target.DataID, target.Slot)
		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		allowMultipleRatings := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagModifiable)) != 0
		roundNegatives := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagRoundMinus)) != 0
		disableSelfRating := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagDisableSelfRating)) != 0

		useMinimum := (ratingSettings.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMin)) != 0
		useMaximum := (ratingSettings.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMax)) != 0

		value := param.RatingValue

		// TODO - The priority of rounding negatives MIGHT be higher than checking the range. Need to verify this
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

		if roundNegatives && value < 0 {
			value = 0
		}

		if disableSelfRating && metaInfo.OwnerID == connection.PID() {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.OperationNotAllowed))
			continue
		}

		if ratingSettings.LockType != types.UInt8(datastore_constants.RatingLockNone) {
			ratingLog, errCode := database.GetUserRatingLog(manager, target.DataID, target.Slot, connection.PID())
			if errCode != nil {
				pRatings = append(pRatings, invalidRatingInfo)
				pResults = append(pResults, types.NewQResult(errCode.ResultCode))
				continue
			}

			if ratingLog.LockExpirationTime != 0 && time.Now().UTC().After(ratingLog.LockExpirationTime.Standard()) {
				pRatings = append(pRatings, invalidRatingInfo)
				pResults = append(pResults, types.NewQResultError(nex.ResultCodes.DataStore.OperationNotAllowed))
				continue
			}
		}

		if !allowMultipleRatings {
			errCode = database.UpdateUserRating(manager, target.DataID, target.Slot, connection.PID(), param.RatingValue)
		} else {
			errCode = database.AddUserRating(manager, target.DataID, target.Slot, connection.PID(), param.RatingValue)
		}

		if errCode != nil {
			pRatings = append(pRatings, invalidRatingInfo)
			pResults = append(pResults, types.NewQResult(errCode.ResultCode))
			continue
		}

		if ratingSettings.LockType != types.UInt8(datastore_constants.RatingLockNone) {
			errCode := database.LockUserRatings(manager, target.DataID, target.Slot, connection.PID(), ratingSettings)
			if errCode != nil {
				pRatings = append(pRatings, invalidRatingInfo)
				pResults = append(pResults, types.NewQResult(errCode.ResultCode))
				continue
			}
		}

		if fetchRatings {
			pRating, errCode := database.GetObjectRatingByDataIDAndSlot(manager, target.DataID, target.Slot)
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

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pRatings.WriteTo(rmcResponseStream)
	pResults.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObjects
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
