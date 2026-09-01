package matchmake_referee_database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"
)

func EndRound(manager *common_globals.UtilityManager, pid types.PID, roundId types.UInt64, personalRoundResults types.List[matchmake_referee_types.MatchmakeRefereePersonalRoundResult]) *nex.Error {

	// * Initialize category that will be fetched for stats updating

	var category types.UInt64

	// TODO - Are the states supposed to have a specific value, or is it server-assigned?

	err := manager.Database.QueryRow(`UPDATE matchmake_referee.rounds SET state = 1 where round_id = $1 RETURNING personal_data_category`, roundId).Scan(&category)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
	}

	for _, result := range personalRoundResults {
		_, err := manager.Database.Exec(`
			INSERT INTO matchmake_referee.round_results
				(round_id, pid, personal_round_result_flag, round_win_loss,
				rating_value_change, buffer) 
				VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (round_id, pid) DO NOTHING /* Result was already inserted by another RMC call*/`, roundId, result.PID, result.PersonalRoundResultFlag, result.RoundWinLoss, result.RatingValueChange, result.Buffer)
		if err != nil {
			common_globals.Logger.Error(err.Error())
			return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}

		// * While it may seem beneficial to wrap the whole function to only run if it is the connection PID
		// * Players that get disconnect/leave with EndRoundWithoutReport still have a report from other players. The above is run by everyone so those reports are fetched.

		if pid == result.PID {
			_, err := manager.Database.Exec(`
				UPDATE matchmake_referee.stats SET rating_value = rating_value + $1 WHERE category = $2 AND owner_pid = $3`, result.RatingValueChange, category, result.PID)
			if err != nil {
				common_globals.Logger.Error(err.Error())
				return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
			}

			switch result.RoundWinLoss {
			case 0:
				_, err := manager.Database.Exec(`
					UPDATE matchmake_referee.stats SET recent_loss = recent_loss + 1, total_loss = total_loss + 1 WHERE category = $1 AND owner_pid = $2`, category, result.PID)
				if err != nil {
					return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
				}
			case 1:
				_, err := manager.Database.Exec(`
					UPDATE matchmake_referee.stats SET recent_win = recent_win + 1, total_win = total_win + 1 WHERE category = $1 AND owner_pid = $2`, category, result.PID)
				if err != nil {
					return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
				}
			case 2:
				_, err := manager.Database.Exec(`
					UPDATE matchmake_referee.stats SET recent_draw = recent_draw + 1, total_draw = total_draw + 1 WHERE category = $1 AND owner_pid = $2`, category, result.PID)
				if err != nil {
					return nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
				}
			default:
				common_globals.Logger.Errorf("Unknown RoundWinLoss state: %d", result.RoundWinLoss)
				return nex.NewError(nex.ResultCodes.Core.Unknown, "Unknown RoundWinLoss state")
			}
		}
	}
	return nil
}
