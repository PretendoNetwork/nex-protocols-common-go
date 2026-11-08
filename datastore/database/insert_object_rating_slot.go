package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func insertObjectRatingSlot(database common_globals.Database, dataID uint64, ratingInitParam datastore_types.DataStoreRatingInitParamWithSlot) *nex.Error {
	allowMultipleRatings := ratingInitParam.Param.Flag.HasFlag(datastore_constants.RatingFlagModifiable)
	roundNegatives := ratingInitParam.Param.Flag.HasFlag(datastore_constants.RatingFlagRoundMinus)
	disableSelfRating := ratingInitParam.Param.Flag.HasFlag(datastore_constants.RatingFlagDisableSelfRating)

	useMinimum := ratingInitParam.Param.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMin)
	useMaximum := ratingInitParam.Param.InternalFlag.HasFlag(datastore_constants.RatingInternalFlagUseRangeMax)

	_, err := database.Exec(`INSERT INTO datastore.rating_slots (
		data_id,
		slot,
		raw_flags,
		raw_internal_flags,
		minimum_value,
		maximum_value,
		initial_value,
		total_value,
		lock_type,
		lock_period_duration,
		lock_period_hour,
		allow_multiple_ratings,
		round_negatives,
		disable_self_rating,
		use_minimum,
		use_maximum
	) VALUES (
		$1,
		$2,
		$3,
		$4,
		$5,
		$6,
		$7,
		$7,
		$8,
		$9,
		$10,
		$11,
		$12,
		$13,
		$14,
		$15
	) ON CONFLICT (data_id, slot) DO NOTHING`,
		dataID,
		ratingInitParam.Slot,
		ratingInitParam.Param.Flag,
		ratingInitParam.Param.InternalFlag,
		ratingInitParam.Param.RangeMin,
		ratingInitParam.Param.RangeMax,
		ratingInitParam.Param.InitialValue,
		ratingInitParam.Param.LockType,
		ratingInitParam.Param.PeriodDuration,
		ratingInitParam.Param.PeriodHour,
		allowMultipleRatings,
		roundNegatives,
		disableSelfRating,
		useMinimum,
		useMaximum,
	)

	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
