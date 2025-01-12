package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
)

// UnregisterGathering unregisters a given gathering on a database
func UnregisterGathering(manager *common_globals.MatchmakingManager, sourcePID types.PID, id uint32) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.gatherings SET registered=false WHERE id=$1`, id)
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

	nexError := tracking.LogUnregisterGathering(manager.Database, sourcePID, id)
	if nexError != nil {
		return nexError
	}

	return nil
}
