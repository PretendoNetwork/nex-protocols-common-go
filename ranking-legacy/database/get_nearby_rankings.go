package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
)

func GetNearbyRankings(manager *commonglobals.RankingManager, pid types.PID, uniqueID types.UInt32, category types.UInt32, orderParam rankinglegacytypes.RankingOrderParam, length types.UInt8) (types.List[rankinglegacytypes.RankingData], *nex.Error) {
	result := make(types.List[rankinglegacytypes.RankingData], 0, length)

	rows, err := manager.Database.Query(`
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
		),
		/* Find our own ranking, then its ordinal +- limit. */
		central_user AS (
		    SELECT
		    	ord,
		    	/* Work out ord +- limit with saturation, $7: limit */
		    	GREATEST(ord - $7 / 2, 1) AS min_ord,
		    	LEAST(ord + $7 / 2, (SELECT MAX(ord) FROM ranking)) AS max_ord
		    FROM ranking
		    WHERE owner_pid = $5 AND unique_id = $6
		)
		/* Select the scores in that ordinal range. */
		SELECT
		    r.owner_pid, r.unique_id, r.category, r.scores, r.unk1, r.unk2,
		    /* CommonData. TODO: NEX 1 allocation behaviour */
		    COALESCE(c.data, ''::bytea),
		    r.ord
		FROM central_user, ranking r LEFT JOIN ranking_legacy.common_data c
		    ON c.owner_pid = r.owner_pid AND c.unique_id = r.unique_id
		WHERE 
		    r.ord >= central_user.min_ord AND
		    r.ord <= central_user.max_ord
		ORDER BY r.ord
		LIMIT $7
	`,
		category,
		orderParam.RankCalculation == 0,
		/* True for golf scoring (lower is better) */
		orderParam.ScoreOrder == 0,
		/* Arrays are one-based in Postgres..? */
		orderParam.ScoreIndex+1,
		pid,
		uniqueID,
		length,
	)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	defer rows.Close()

	for rows.Next() {
		score := rankinglegacytypes.NewRankingData()

		// WORKAROUND: for some reason, types.Buffer.Scan can return duplicate values. UaF?
		var commonData []byte

		err := rows.Scan(
			&score.PrincipalID,
			&score.UniqueID,
			&score.Category,
			&score.Scores,
			&score.Unknown1,
			&score.Unknown2,
			&commonData,
			&score.Order,
		)
		if err != nil {
			commonglobals.Logger.Error(err.Error())
			return nil, nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
		}
		score.CommonData = commonData

		result = append(result, score)
	}

	return result, nil
}
