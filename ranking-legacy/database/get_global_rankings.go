package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
)

func GetGlobalRankings(manager *commonglobals.RankingManager, category types.UInt32, orderParam rankinglegacytypes.RankingOrderParam, offset types.UInt32, length types.UInt8) (types.List[rankinglegacytypes.RankingData], *nex.Error) {
	result := make(types.List[rankinglegacytypes.RankingData], 0, length)

	rows, err := manager.Database.Query(`
		SELECT
		    s.owner_pid, s.unique_id, s.category, s.scores, s.unk1, s.unk2,
		    /* CommonData. TODO: NEX 1 allocation behaviour */
		    COALESCE(c.data, ''::bytea),
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
		
		FROM ranking_legacy.scores s LEFT JOIN ranking_legacy.common_data c
		    ON c.owner_pid = s.owner_pid AND c.unique_id = s.unique_id
		WHERE s.category = $1 AND s.deleted = false
		ORDER BY ord
		OFFSET $5
		LIMIT $6
	`,
		category,
		orderParam.RankCalculation == 0,
		/* True for golf scoring (lower is better) */
		orderParam.ScoreOrder == 0,
		/* Arrays are one-based in Postgres..? */
		orderParam.ScoreIndex+1,
		offset, length,
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
