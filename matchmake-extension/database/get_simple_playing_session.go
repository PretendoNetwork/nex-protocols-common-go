package database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// GetSimplePlayingSession returns the simple playing sessions of the given PIDs
func GetSimplePlayingSession(manager *common_globals.MatchmakingManager, listPID []types.PID) ([]match_making_types.SimplePlayingSession, *nex.Error) {
	simplePlayingSessions := make([]match_making_types.SimplePlayingSession, 0)
	for _, pid := range listPID {
		simplePlayingSession := match_making_types.NewSimplePlayingSession()

		err := manager.Database.QueryRow(`SELECT
		g.id,
		ms.attribs[1],
		ms.game_mode
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		$1=ANY(g.participants)`, pid).Scan(
		&simplePlayingSession.GatheringID,
		&simplePlayingSession.Attribute0,
		&simplePlayingSession.GameMode)
		if err != nil {
			if err != sql.ErrNoRows {
				common_globals.Logger.Critical(err.Error())
			}
			continue
		}

		simplePlayingSession.PrincipalID = pid

		simplePlayingSessions = append(simplePlayingSessions, simplePlayingSession)
	}

	return simplePlayingSessions, nil
}
