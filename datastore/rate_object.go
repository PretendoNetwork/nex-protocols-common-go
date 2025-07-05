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

func (commonProtocol *CommonProtocol) rateObject(err error, packet nex.PacketInterface, callID uint32, target datastore_types.DataStoreRatingTarget, param datastore_types.DataStoreRateObjectParam, fetchRatings types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, "change_error")
	}

	manager := commonProtocol.manager
	connection := packet.Sender()
	endpoint := connection.Endpoint()

	// TODO - Add rollback for when error occurs

	if target.DataID == types.UInt64(datastore_constants.InvalidDataID) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if target.Slot >= types.UInt8(datastore_constants.NumRatingSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager, target.DataID)
	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectAccessPermission(*manager, connection.PID(), metaInfo, accessPassword, param.AccessPassword)
	if errCode != nil {
		return nil, errCode
	}

	// * The owner of an object can always view their objects, but normal users cannot
	if metaInfo.Status != types.UInt8(datastore_constants.DataStatusNone) && metaInfo.OwnerID != connection.PID() {
		if metaInfo.Status == types.UInt8(datastore_constants.DataStatusPending) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
		}

		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	// TODO - Refactor out this hack. We store decoded flags in Postgres, stop manually parsing them again here
	ratingSettings, errCode := database.GetObjectRatingSlotSettings(manager, target.DataID, target.Slot)
	if errCode != nil {
		return nil, errCode
	}

	allowMultipleRatings := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagModifiable)) != 0
	roundNegatives := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagRoundMinus)) != 0
	disableSelfRating := (ratingSettings.Flag & types.UInt8(datastore_constants.RatingFlagDisableSelfRating)) != 0

	useMinimum := (ratingSettings.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMin)) != 0
	useMaximum := (ratingSettings.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMax)) != 0

	value := param.RatingValue

	// TODO - The priority of rounding negatives MIGHT be higher than checking the range. Need to verify this
	if useMinimum && value < ratingSettings.RangeMin {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if useMaximum && value > ratingSettings.RangeMax {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if roundNegatives && value < 0 {
		value = 0
	}

	if disableSelfRating && metaInfo.OwnerID == connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	if ratingSettings.LockType != types.UInt8(datastore_constants.RatingLockNone) {
		ratingLog, errCode := database.GetUserRatingLog(manager, target.DataID, target.Slot, connection.PID())
		if errCode != nil {
			return nil, errCode
		}

		if ratingLog.LockExpirationTime != 0 && time.Now().UTC().After(ratingLog.LockExpirationTime.Standard()) {
			return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
		}
	}

	if !allowMultipleRatings {
		errCode = database.UpdateUserRating(manager, target.DataID, target.Slot, connection.PID(), param.RatingValue)
	} else {
		errCode = database.AddUserRating(manager, target.DataID, target.Slot, connection.PID(), param.RatingValue)
	}

	if errCode != nil {
		return nil, errCode
	}

	if ratingSettings.LockType != types.UInt8(datastore_constants.RatingLockNone) {
		errCode := database.LockUserRatings(manager, target.DataID, target.Slot, connection.PID(), ratingSettings)
		if errCode != nil {
			return nil, errCode
		}
	}

	// TODO - Create a log here manually, or continue to manually create logs at runtime?

	var pRating datastore_types.DataStoreRatingInfo

	if fetchRatings {
		pRating, errCode = database.GetObjectRatingByDataIDAndSlot(manager, target.DataID, target.Slot)
		if errCode != nil {
			return nil, errCode
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	pRating.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = datastore.ProtocolID
	rmcResponse.MethodID = datastore.MethodRateObject
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
