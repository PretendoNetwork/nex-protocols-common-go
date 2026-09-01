package matchmake_referee_database

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"
)

func GetOrCreateStats(manager *common_globals.UtilityManager, pid types.PID, category types.UInt32, initialRatingValue types.UInt32) (matchmake_referee_types.MatchmakeRefereeStats, *nex.Error) {
	stats := matchmake_referee_types.NewMatchmakeRefereeStats()

	// * We need the Unique ID of the player for the query. However, the client never sends the Unique ID inside of their request
	// * To get around this, we check the user's Primary Unique ID based on their PID through the Utility database.
	// * Mini version of CheckUserHasPrimaryUniqueID

	// ? 08/04/2026 : Kirby Battle Royale appears to be creating a new unique ID every time it connects? I'm unsure as to why this is occurring, but this is flooding the stats table
	// ? Try to look into what is going on, or look for a different method of fetching the Unique ID

	// ? 08/05/2026 : Smash does not ever call for a unique ID, yet one is expected to be generated. This matches up with Kirby Fighters 2, where a unique ID is given despite there not being a call
	// ? Adding a function to generate a unique ID to account for this for now

	var unique_id types.UInt64

	err := manager.Database.QueryRow(`SELECT unique_id FROM utility.unique_ids WHERE associated_pid = $1 AND is_primary_id = true`, pid).Scan(&unique_id)
	if err != nil {
		if err == sql.ErrNoRows {
			// * User does not have a unique ID. To account for this, we will create one for them.

			uniqueIDInfo, nexError := manager.GenerateNEXUniqueIDWithPassword(manager, pid, false)
			if nexError != nil {
				common_globals.Logger.Error(nexError.Error())
				return stats, nexError
			}
			unique_id = uniqueIDInfo.NEXUniqueID
		} else {
			common_globals.Logger.Error(err.Error())
			return stats, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
		}
	}
	row := manager.Database.QueryRow(`
        SELECT unique_id, category, owner_pid,
            recent_disconnection, recent_violation,
            recent_mismatch, recent_win,
            recent_loss, recent_draw,
            total_disconnect, total_violation, 
			total_mismatch, total_win, total_loss,
            total_draw, rating_value
        FROM matchmake_referee.stats WHERE unique_Id = $1 AND category = $2 AND owner_pid = $3`, unique_id, category, pid)

	err = row.Scan(&stats.UniqueID, &stats.Category, &stats.PID, &stats.RecentDisconnection, &stats.RecentViolation,
		&stats.RecentMismatch, &stats.RecentWin, &stats.RecentLoss, &stats.RecentDraw,
		&stats.TotalDisconnect, &stats.TotalViolation, &stats.TotalMismatch, &stats.TotalWin, &stats.TotalLoss,
		&stats.TotalDraw, &stats.RatingValue)
	if err != nil {
		if err == sql.ErrNoRows {
			//No row was found, meaning the user does not have existing stats. We need to create them.

			_, err := manager.Database.Exec(`
           INSERT INTO matchmake_referee.stats (
               unique_id, category, owner_pid,
               recent_disconnection, recent_violation,
               recent_mismatch, recent_win,
               recent_loss, recent_draw,
               total_disconnect, total_violation,
				total_mismatch, total_win, total_loss,
               total_draw, rating_value
           ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
       `, unique_id, category, pid, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, initialRatingValue)
			if err != nil {
				common_globals.Logger.Error(err.Error())
				return stats, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
			}

			stats.UniqueID = unique_id
			stats.Category = category
			stats.PID = pid
			stats.RecentDisconnection = 0
			stats.RecentViolation = 0
			stats.RecentMismatch = 0
			stats.RecentWin = 0
			stats.RecentLoss = 0
			stats.RecentDraw = 0
			stats.TotalDisconnect = 0
			stats.TotalViolation = 0
			stats.TotalMismatch = 0
			stats.TotalWin = 0
			stats.TotalLoss = 0
			stats.TotalDraw = 0
			stats.RatingValue = initialRatingValue

			return stats, nil
		} else {
			common_globals.Logger.Error(err.Error())
			return stats, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
		}
	}

	return stats, nil
}
