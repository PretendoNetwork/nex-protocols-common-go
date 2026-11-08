package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func LockRatingSlotsByDataStoreRatingTargets(database common_globals.Database, targets types.List[datastore_types.DataStoreRatingTarget]) *nex.Error {
	dataIDs := make([]types.UInt64, 0, len(targets))
	slots := make([]types.UInt8, 0, len(targets))

	for _, target := range targets {
		dataIDs = append(dataIDs, target.DataID)
		slots = append(slots, target.Slot)
	}

	_, err := database.Exec(`
		SELECT 1
		FROM datastore.rating_slots
		WHERE (data_id, slot) IN (SELECT * FROM unnest($1::numeric[], $2::smallint[]))
		ORDER BY data_id, slot
		FOR UPDATE
	`, pq.Array(dataIDs), pq.Array(slots))
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
