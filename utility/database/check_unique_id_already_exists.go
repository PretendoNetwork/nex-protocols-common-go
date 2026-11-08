package utility_database

import (
	"database/sql"
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// CheckUniqueIDAlreadyExists takes a UniqueIDInfo and returns an error if the unique ID is already present in the database
func CheckUniqueIDAlreadyExists(manager *common_globals.UtilityManager, uniqueIDInfo utility_types.UniqueIDInfo) *nex.Error {
	var uniqueID types.UInt64

	err := manager.Database.QueryRow(`SELECT unique_id FROM utility.unique_ids WHERE unique_id=$1`,
		uniqueIDInfo.NEXUniqueID,
	).Scan(
		&uniqueID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	return nex.NewError(nex.ResultCodes.Core.SystemError, fmt.Sprintf("Unique ID (%d) already exists", uniqueID))
}
