package database

import (
	"database/sql"
	"errors"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
)

func GetOwnRanking(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32, category types.UInt32, orderParam rankinglegacytypes.RankingOrderParam) (types.UInt32, *nex.Error) {
	var result types.UInt32
	err := manager.Database.QueryRow(`
		/* Do score calculations for the category */
		WITH ranking AS (
		    SELECT *,
			/* $2: RankCalculation */
			CASE WHEN $2 THEN
		        RANK() OVER (ORDER BY 
		            /* $3: ScoreOrder (golf scoring), $4: ScoreIndex */
		            CASE WHEN $3 THEN s.scores[$4] END ASC,
					CASE WHEN NOT $3 THEN s.scores[$4] END DESC
				)
			ELSE
			    ROW_NUMBER() OVER (ORDER BY 
		            CASE WHEN $3 THEN s.scores[$4] END ASC,
					CASE WHEN NOT $3 THEN s.scores[$4] END DESC
				)
			END AS ord
			
		    FROM ranking_legacy.scores s
		    WHERE
		        s.category = $1 AND s.deleted = false
		)
		/* Find our own ranking */
		SELECT ord FROM ranking
		WHERE owner_pid = $5 AND unique_id = $6
	`,
		category,
		orderParam.RankCalculation == 0,
		/* True for golf scoring (lower is better) */
		orderParam.ScoreOrder == 0,
		/* Arrays are one-based in Postgres..? */
		orderParam.ScoreIndex+1,
		pid,
		uniqueID,
	).Scan(&result)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nex.NewError(nex.ResultCodes.Ranking.NotFound, "Own ranking not found!")
	} else if err != nil {
		commonglobals.Logger.Error(err.Error())
		return 0, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}

	return result, nil
}
