package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateProgressScore updates the progress score on a matchmake session
func UpdateProgressScore(manager *common_globals.MatchmakingManager, gatheringID uint32, progressScore uint8) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET progress_score=$1 WHERE id=$2`, progressScore, gatheringID)
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
