package utility_database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// CheckCanAssociateUniqueIDs handles all relevant validity checking (including the existence of a unique ID, if the password matches, etc)
func CheckCanAssociateUniqueIDs(manager *common_globals.UtilityManager, userPID types.PID, uniqueIDInfos types.List[utility_types.UniqueIDInfo]) *nex.Error {
	var uniqueID, dbPassword types.UInt64
	var associatedPID types.PID

	uniqueIDs := make([]types.UInt64, 0)
	for _, uniqueIDInfo := range uniqueIDInfos {
		uniqueIDs = append(uniqueIDs, uniqueIDInfo.NEXUniqueID)
	}

	rows, err := manager.Database.Query(`SELECT unique_id, associated_pid, password FROM utility.unique_ids WHERE unique_id=ANY($1)`,
		uniqueIDs,
	)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}
	defer rows.Close()

	rowCount := 0
	for rows.Next() {
		rowCount++

		err = rows.Scan(
			&uniqueID,
			&associatedPID,
			&dbPassword,
		)
		if err != nil {
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		var targetConnection *nex.PRUDPConnection = nil
		if associatedPID != 0 {
			targetConnection = manager.Endpoint.FindConnectionByPID(uint64(associatedPID))

			if !manager.AllowUniqueIDStealing && associatedPID != userPID || targetConnection != nil && associatedPID != userPID {
				return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Unique ID stealing is disabled or the owner is online")
			}
		}

		index := slices.Index(uniqueIDs, uniqueID)
		if index == -1 {
			return nex.NewError(nex.ResultCodes.Core.Unknown, "Index of unique ID not found in array, this SHOULD NOT HAPPEN")
		}

		if dbPassword != uniqueIDInfos[index].NEXUniqueIDPassword {
			return nex.NewError(nex.ResultCodes.Core.AccessDenied, "Invalid password for a unique ID")
		}
	}

	if rowCount != len(uniqueIDs) {
		return nex.NewError(nex.ResultCodes.Core.InvalidArgument, "One or more of the provided unique IDs do not exist")
	}

	return nil
}
