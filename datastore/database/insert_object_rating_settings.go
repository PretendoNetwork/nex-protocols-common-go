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
		errCode := validateRatingInitParam(ratingInitParam)
		if errCode != nil {
			return errCode
		}
	}

	// TODO - Handle rollback on errors
	for _, ratingInitParam := range ratingInitParams {
		errCode := insertObjectRatingSlot(database, dataID, ratingInitParam)
		if errCode != nil {
			return errCode
		}
	}

	return nil
}
