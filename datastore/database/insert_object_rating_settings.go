package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func InsertObjectRatingSettings(manager *common_globals.DataStoreManager, dataID uint64, ratingInitParams types.List[datastore_types.DataStoreRatingInitParamWithSlot]) *nex.Error {
	if len(ratingInitParams) > int(datastore_constants.NumRatingSlot) {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with too many RatingInitParams")
	}

	for _, ratingInitParam := range ratingInitParams {
		// * Sent as an SInt8, but only values 0-15 are allowed? Why Nintendo?
		if ratingInitParam.Slot < 0 || ratingInitParam.Slot > types.Int8(datastore_constants.RatingSlotMax) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating slot")
		}

		if ratingInitParam.Param.LockType > types.UInt8(datastore_constants.RatingLockPermanent) {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid rating lock type")
		}

		// * Different lock types interpret the lock values differently

		// * "Interval" locks treat PeriodDuration as a non-negative value representing
		// * the number of seconds until the lock expires
		if ratingInitParam.Param.LockType == types.UInt8(datastore_constants.RatingLockInterval) {
			if ratingInitParam.Param.PeriodDuration < 0 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}
		}

		// * "Period" locks treat PeriodDuration as the day of the week/month and
		// * PeriodHour as the hour of that day. "Day1" is the first of the following
		// * month
		if ratingInitParam.Param.LockType == types.UInt8(datastore_constants.RatingLockPeriod) {
			if ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodMon) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodTue) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodWed) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodThu) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodFri) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodSat) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodSun) &&
				ratingInitParam.Param.PeriodDuration != types.Int16(datastore_constants.RatingLockPeriodDay1) {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodDuration")
			}

			// * Sent as an SInt8, I wonder if "negative" time is possible?
			// * Like for referencing days in the past? This would allow the
			// * client to target the LAST day of the month as well as the first?
			if ratingInitParam.Param.PeriodHour < 0 || ratingInitParam.Param.PeriodHour > 23 {
				return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "Tried to upload object with an invalid PeriodHour")
			}
		}
	}

	// TODO - Handle rollback on errors
	for _, ratingInitParam := range ratingInitParams {
		allowMultipleRatings := (ratingInitParam.Param.Flag & types.UInt8(datastore_constants.RatingFlagModifiable)) != 0
		roundNegatives := (ratingInitParam.Param.Flag & types.UInt8(datastore_constants.RatingFlagRoundMinus)) != 0
		disableSelfRating := (ratingInitParam.Param.Flag & types.UInt8(datastore_constants.RatingFlagDisableSelfRating)) != 0

		useMinimum := (ratingInitParam.Param.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMin)) != 0
		useMaximum := (ratingInitParam.Param.InternalFlag & types.UInt8(datastore_constants.RatingInternalFlagUseRangeMax)) != 0

		_, err := manager.Database.Exec(`INSERT INTO datastore.rating_settings (
			data_id,
			slot,
			raw_flags,
			raw_internal_flags,
			minimum_value,
			maximum_value,
			initial_value,
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
