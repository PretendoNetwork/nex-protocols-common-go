package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateParticipation updates the participation of a matchmake session
func UpdateParticipation(manager *common_globals.MatchmakingManager, gatheringID uint32, participation bool) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET open_participation=$1 WHERE id=$2`, participation, gatheringID)
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	if rowsAffected == 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	return nil
}
