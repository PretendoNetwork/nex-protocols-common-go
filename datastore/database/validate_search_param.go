package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func validateSearchParam(callerPID types.PID, param datastore_types.DataStoreSearchParam) *nex.Error {
	dates := []types.DateTime{
		param.CreatedAfter,
		param.CreatedBefore,
		param.UpdatedAfter,
		param.UpdatedBefore,
	}

	for _, date := range dates {
		if date.IsNever() || date.IsInvalidDateTime() {
			continue
		}

		// * Dirty hack to see if a date is real or not, since converting round trip
		// * will create different dates if the date isn't real
		var roundTrip types.DateTime
		if date.Year() < 1 || date.Year() > 9999 || roundTrip.FromTimestamp(date.Standard()) != date {
			return nex.NewError(nex.ResultCodes.PythonCore.ConversionError, "Tried to search with an invalid date")
		}
	}

	if param.SearchTarget < datastore_constants.SearchTypePublic || param.SearchTarget > datastore_constants.SearchTypeOwnAll {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with an invalid search type")
	}

	if param.OwnerType > datastore_constants.SearchTargetAnybodyExcludeSpecified {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with an invalid owner type")
	}

	if param.OwnerType == datastore_constants.SearchTargetFriend {
		if param.SearchTarget != datastore_constants.SearchTypePublic && param.SearchTarget != datastore_constants.SearchTypeFriend && param.SearchTarget != datastore_constants.SearchTypeReceivedSpecified && param.SearchTarget != datastore_constants.SearchTypeReceived {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for friends with a search type which cannot be limited to friends")
		}

		if len(param.OwnerIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for friends and owner IDs")
		}
	}

	if param.SearchTarget == datastore_constants.SearchTypeOwn ||
		param.SearchTarget == datastore_constants.SearchTypeOwnPending ||
		param.SearchTarget == datastore_constants.SearchTypeOwnRejected ||
		param.SearchTarget == datastore_constants.SearchTypeOwnAll {
		if len(param.OwnerIDs) != 0 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for the callers own objects and owner IDs")
		}
	}

	if param.SearchTarget == datastore_constants.SearchTypeReceivedSpecified || param.SearchTarget == datastore_constants.SearchTypeReceived {
		if len(param.DestinationIDs) > 1 || (len(param.DestinationIDs) == 1 && param.DestinationIDs[0] != callerPID) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for received objects with destination IDs other than the caller")
		}
	}

	dataTypeCount := len(param.DataTypes)
	if param.DataType != types.UInt16(datastore_constants.InvalidDataType) {
		dataTypeCount++
	}

	if dataTypeCount > datastore_constants.MaxSearchDataTypeSize {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for more than MaxSearchDataTypeSize data types")
	}

	if getSearchDateFilter(param.CreatedAfter) != nil && getSearchDateFilter(param.CreatedBefore) != nil && param.CreatedAfter > param.CreatedBefore {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for a created range which ends before it starts")
	}

	if getSearchDateFilter(param.UpdatedAfter) != nil && getSearchDateFilter(param.UpdatedBefore) != nil && param.UpdatedAfter > param.UpdatedBefore {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for an updated range which ends before it starts")
	}

	if len(param.Tags) > int(datastore_constants.NumTagSlot) {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search for too many tags")
	}

	sortByRating := param.ResultOrderColumn >= datastore_constants.SearchSortColumnRating0 && param.ResultOrderColumn <= datastore_constants.SearchSortColumnRating15
	sortByRatingAverage := param.ResultOrderColumn >= datastore_constants.SearchSortColumnRatingAverage0 && param.ResultOrderColumn <= datastore_constants.SearchSortColumnRatingAverage15

	if param.ResultOrderColumn > datastore_constants.SearchSortColumnOwnerID && !sortByRating && !sortByRatingAverage {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with an invalid sort column")
	}

	if param.ResultOrder > datastore_constants.SearchSortOrderDesc {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with an invalid sort order")
	}

	maxResultSize := datastore_constants.MaxSearchResultSize
	if uint64(param.ResultRange.Offset) == datastore_constants.ResultRangeAnyOffset {
		maxResultSize = datastore_constants.MaxSearchAnyResultSize
	}

	if param.ResultRange.Length == 0 || int(param.ResultRange.Length) > maxResultSize {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with an invalid result range")
	}

	if param.MinimalRatingFrequency != 0 && !sortByRating && !sortByRatingAverage {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to search with a minimal rating frequency but not a rating column")
	}

	return nil
}
