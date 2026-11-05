package database

import (
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func InsertCachedSearchObjectResponse(database common_globals.Database, manager *common_globals.DataStoreManager, pid types.PID, cacheKey []byte, response []byte) *nex.Error {
	// * Cache forced disabled, even if the client wants to use it
	if manager.SearchCacheLifetime <= 0 {
		return nil
	}

	// * Get rid of old expired caches
	_, err := database.Exec(`DELETE FROM datastore.search_cache WHERE expiration_date <= (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')`)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	_, err = database.Exec(`
		INSERT INTO datastore.search_cache (
			pid,
			param,
			response,
			expiration_date
		) VALUES (
			$1,
			$2,
			$3,
			$4
		) ON CONFLICT (pid, param) DO UPDATE
		SET response = EXCLUDED.response, expiration_date = EXCLUDED.expiration_date
	`,
		pid,
		cacheKey,
		response,
		time.Now().UTC().Add(manager.SearchCacheLifetime),
	)
	if err != nil {
		// TODO - Send more specific errors?
		return nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return nil
}
