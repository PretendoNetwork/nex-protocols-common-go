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
		return nil, nex.NewError(nex.ResultCodes.DataStore.NotFound, "change_error")
	}

	if target.Slot >= types.UInt8(datastore_constants.NumRatingSlot) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	metaInfo, accessPassword, errCode := database.GetAccessObjectInfoByDataID(manager.Database, target.DataID)
	if errCode != nil {
		return nil, errCode
	}

	errCode = manager.VerifyObjectAccessPermission(connection.PID(), metaInfo, accessPassword, param.AccessPassword)
	if errCode != nil {
		return nil, errCode
	}

	// * The owner of an object can always view their objects, but normal users cannot
	if metaInfo.Status != datastore_constants.DataStatusNone && metaInfo.OwnerID != connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.UnderReviewing, "change_error")
	}

	// TODO - Refactor out this hack. We store decoded flags in Postgres, stop manually parsing them again here
	ratingSettings, errCode := database.GetObjectRatingSlotSettings(manager.Database, target.DataID, target.Slot)
	if errCode != nil {
		return nil, errCode
	}

	// * The unknown flags 0x01, 0x02 and 0x20 make the slot impossible to rate.
	// * 0x01 crashes on official servers, while 0x02 and 0x20 are just refused
	// TODO - Move these unknown flags to nex-protocols-go

	if ratingSettings.Flag.HasFlag(0x01) {
		return nil, nex.NewError(nex.ResultCodes.PythonCore.Exception, "change_error")
	}

	if ratingSettings.Flag.HasFlag(0x02) || ratingSettings.Flag.HasFlag(0x20) {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	disableSelfRating := ratingSettings.Flag.HasFlag(datastore_constants.RatingFlagDisableSelfRating)

	useMinimum := ratingSettings.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMin)
	useMaximum := ratingSettings.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMax)

	value := param.RatingValue

	if useMinimum && value < ratingSettings.RangeMin {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if useMaximum && value > ratingSettings.RangeMax {
		return nil, nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	if disableSelfRating && metaInfo.OwnerID == connection.PID() {
		return nil, nex.NewError(nex.ResultCodes.DataStore.OperationNotAllowed, "change_error")
	}

	errCode = database.RateObjectSlot(manager.Database, manager, target.DataID, target.Slot, connection.PID(), value, ratingSettings)
	if errCode != nil {
		return nil, errCode
	}

	// TODO - Create a log here manually, or continue to manually create logs at runtime?

	var pRating datastore_types.DataStoreRatingInfo

	if fetchRatings {
		pRating, errCode = database.GetObjectRatingByDataIDAndSlot(manager.Database, target.DataID, target.Slot)
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
