package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/lib/pq"
)

func LockRatingSlotsByDataIDs(database common_globals.Database, dataIDs types.List[types.UInt64]) *nex.Error {
	_, err := database.Exec(`
		SELECT 1
		FROM datastore.rating_slots
		WHERE data_id = ANY($1::numeric[])
		ORDER BY data_id, slot
		FOR UPDATE
	`, pq.Array(dataIDs))
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
