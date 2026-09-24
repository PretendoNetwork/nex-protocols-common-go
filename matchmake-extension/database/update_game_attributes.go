package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// UpdateGameAttributes updates all attributes on a matchmake session
func UpdateGameAttributes(manager *common_globals.MatchmakingManager, gatheringID uint32, attributes types.List[types.UInt32]) *nex.Error {
	attribs := make([]uint32, len(attributes))
	for i, value := range attributes {
		attribs[i] = uint32(value)
	}

	result, err := manager.Database.Exec(`UPDATE matchmaking.matchmake_sessions SET attribs=$1 WHERE id=$2`, attributes, gatheringID)
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
