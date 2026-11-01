package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func InsertObjectRatingSettings(database common_globals.Database, dataID uint64, ratingInitParams types.List[datastore_types.DataStoreRatingInitParamWithSlot]) *nex.Error {
	if len(ratingInitParams) > int(datastore_constants.NumRatingSlot) {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many RatingInitParams")
	}

	for _, ratingInitParam := range ratingInitParams {
		// * Sent as an SInt8, but only values 0-15 are allowed? Why Nintendo?
		if ratingInitParam.Slot < 0 || ratingInitParam.Slot > types.Int8(datastore_constants.RatingSlotMax) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating slot")
		}

		if ratingInitParam.Param.LockType > datastore_constants.RatingLockPermanent {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating lock type")
		}

		// * Different lock types interpret the lock values differently

		// * "Interval" locks treat PeriodDuration as a non-negative value representing
		// * the number of seconds until the lock expires
		if ratingInitParam.Param.LockType == datastore_constants.RatingLockInterval {
			if ratingInitParam.Param.PeriodDuration < 0 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}
		}

		// * "Period" locks treat PeriodDuration as either a day of the week/month, or
		// * a number of days from 0 to 365, and PeriodHour as the hour of that day,
		// * with a negative PeriodHour going into the previous day. "Day1" is the first
		// * of the following month
		if ratingInitParam.Param.LockType == datastore_constants.RatingLockPeriod {
			periodDuration := ratingInitParam.Param.PeriodDuration
			isWeekday := periodDuration >= datastore_constants.RatingLockPeriodSun && periodDuration <= datastore_constants.RatingLockPeriodMon
			isNumberOfDays := periodDuration >= 0 && periodDuration <= 365

			if !isWeekday && !isNumberOfDays && periodDuration != datastore_constants.RatingLockPeriodDay1 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}

			if ratingInitParam.Param.PeriodHour < -23 || ratingInitParam.Param.PeriodHour > 23 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodHour")
			}

			if isNumberOfDays && ratingInitParam.Param.PeriodHour < 0 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodHour")
			}
		}
	}

	// TODO - Handle rollback on errors
	for _, ratingInitParam := range ratingInitParams {
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
		)`,
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
	}

	return nil
}
