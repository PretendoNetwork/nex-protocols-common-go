package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

// TODO - This is mostly a hack for right now, abusing an existing struct
func GetObjectRatingSlotSettings(manager *common_globals.DataStoreManager, dataID types.UInt64, slot types.UInt8) (datastore_types.DataStoreRatingInitParam, *nex.Error) {
	var ratingSettings datastore_types.DataStoreRatingInitParam

	err := manager.Database.QueryRow(`
		SELECT
			raw_flags,
			raw_internal_flags,
			lock_type,
			initial_value,
			minimum_value,
			maximum_value,
			lock_period_hour,
			lock_period_duration
		FROM datastore.rating_settings
		WHERE data_id=$1 AND slot=$2
	`,
		dataID,
		slot,
	).Scan(
		&ratingSettings.Flag,
		&ratingSettings.InternalFlag,
		&ratingSettings.LockType,
		&ratingSettings.InitialValue,
		&ratingSettings.RangeMin,
		&ratingSettings.RangeMax,
		&ratingSettings.PeriodHour,
		&ratingSettings.PeriodDuration,
	)
	if err != nil {
		// TODO - Send more specific errors?
		return ratingSettings, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return ratingSettings, nil
}
