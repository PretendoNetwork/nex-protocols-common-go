package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func GetCachedSearchObjectResponse(database common_globals.Database, pid types.PID, encodedParam []byte) ([]byte, *nex.Error) {
	var response []byte

	err := database.QueryRow(`
		SELECT response
		FROM datastore.search_cache
		WHERE pid = $1 AND param = $2 AND expiration_date > (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
	`, pid, encodedParam).Scan(&response)
	if err != nil {
		if err == sql.ErrNoRows {
			// * No cache or its expired
			return nil, nil
		}

		// TODO - Send more specific errors?
		return nil, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	return response, nil
}
