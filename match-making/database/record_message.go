package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// RecordJoinMessage records the join message from a participant into a gathering
func RecordJoinMessage(manager *common_globals.MatchmakingManager, gatheringID uint32, PID types.PID, message string) *nex.Error {
	_, err := manager.Database.Exec(`INSERT INTO matchmaking.join_messages (gathering_id, pid, message) VALUES ($1, $2, $3) ON CONFLICT (gathering_id, pid) DO UPDATE SET message = $3`, gatheringID, PID, message)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return nil
}
