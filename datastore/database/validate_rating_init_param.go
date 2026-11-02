package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func validateRatingInitParam(ratingInitParam datastore_types.DataStoreRatingInitParamWithSlot) *nex.Error {
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

	return nil
}
