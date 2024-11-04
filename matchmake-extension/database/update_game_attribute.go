package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateGameAttribute updates an attribute on a matchmake session
func UpdateGameAttribute(manager *common_globals.MatchmakingManager, gatheringID uint32, attributeIndex uint32, newValue uint32) *nex.Error {
	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET attribs[$1]=$2 WHERE id=$3`, attributeIndex + 1, newValue, gatheringID)
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
