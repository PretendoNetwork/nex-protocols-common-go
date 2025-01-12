package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
)

// EndMatchmakeSessionsParticipation ends participation on all matchmake sessions
func EndMatchmakeSessionsParticipation(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection) {
	rows, err := manager.Database.Query(`SELECT id FROM matchmaking.gatherings WHERE type='MatchmakeSession' AND $1=ANY(participants)`, connection.PID())
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	for rows.Next() {
		var gatheringID uint32
		err = rows.Scan(&gatheringID)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			continue
		}

		database.EndGatheringParticipation(manager, gatheringID, connection, "")
	}

	rows.Close()
}
