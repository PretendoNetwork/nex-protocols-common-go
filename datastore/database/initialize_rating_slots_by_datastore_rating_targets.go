package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func InitializeRatingSlotsByDataStoreRatingTargets(database common_globals.Database, targets types.List[datastore_types.DataStoreRatingTarget], params types.List[datastore_types.DataStorePreparePostParam]) *nex.Error {
	dataIDs := make([]types.UInt64, 0, len(targets))
	slots := make([]types.UInt8, 0, len(targets))

	for _, target := range targets {
		dataIDs = append(dataIDs, target.DataID)
		slots = append(slots, target.Slot)
	}

	// * Trying to initialize the same slots multiple times at once could create a deadlock,
	// * so process them in data_id/slot order
	// TODO - This is likely not correct behavior but I can't test this method on any live servers right now. Revist this
	var positions []int64

	err := database.QueryRow(`
		SELECT COALESCE(array_agg(target.position - 1 ORDER BY target.data_id, target.slot, target.position), '{}')
		FROM unnest($1::numeric[], $2::smallint[]) WITH ORDINALITY AS target(data_id, slot, position)
	`, pq.Array(dataIDs), pq.Array(slots)).Scan(pq.Array(&positions))
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	for _, position := range positions {
		var param datastore_types.DataStorePreparePostParam
		if len(params) == 1 {
			param = params[0]
		} else {
			param = params[position]
		}

		target := targets[position]

		errCode := InitializeObjectRatingSlotByPreparePostParam(database, target.DataID, target.Slot, param)
		if errCode != nil && !errCode.Equals(nex.ResultCodes.DataStore.InvalidArgument) {
			return errCode
		}
	}

	return nil
}
