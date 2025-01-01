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
		var gatheringID uint32
		var attribute0 uint32
		var gameMode uint32

		err := manager.Database.QueryRow(`SELECT
		g.id,
		ms.attribs[1],
		ms.game_mode
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		$1=ANY(g.participants)`, uint64(pid)).Scan(
		&gatheringID,
		&attribute0,
		&gameMode)
		if err != nil {
			if err != sql.ErrNoRows {
				common_globals.Logger.Critical(err.Error())
			}
			continue
		}

		simplePlayingSession := match_making_types.NewSimplePlayingSession()

		simplePlayingSession.PrincipalID = pid
		simplePlayingSession.GatheringID = types.NewUInt32(gatheringID)
		simplePlayingSession.Attribute0 = types.NewUInt32(attribute0)
		simplePlayingSession.GameMode = types.NewUInt32(gameMode)
		simplePlayingSessions = append(simplePlayingSessions, simplePlayingSession)
	}

	return simplePlayingSessions, nil
}
