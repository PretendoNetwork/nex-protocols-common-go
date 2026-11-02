package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func InitializeObjectRatingSlotByPreparePostParam(database common_globals.Database, dataID types.UInt64, slot types.UInt8, param datastore_types.DataStorePreparePostParam) *nex.Error {
	var objectExists bool
	var slotExists bool

	err := database.QueryRow(`
		SELECT
			EXISTS (SELECT 1 FROM datastore.objects WHERE data_id = $1),
			EXISTS (SELECT 1 FROM datastore.rating_slots WHERE data_id = $1 AND slot = $2)
	`, dataID, slot).Scan(&objectExists, &slotExists)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	if !objectExists || slotExists {
		return nil
	}

	for _, ratingInitParam := range param.RatingInitParams {
		if int(ratingInitParam.Slot) != int(slot) {
			continue
		}

		errCode := validateRatingInitParam(ratingInitParam)
		if errCode != nil {
			return errCode
		}

		return insertObjectRatingSlot(database, uint64(dataID), ratingInitParam)
	}

	return nil
}
