package database

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

func UploadScores(manager *commonglobals.RankingManager, uniqueID types.UInt32, scores types.List[rankinglegacytypes.RankingScore], pid types.PID) *nex.Error {
	query := `INSERT INTO ranking_legacy.scores (create_time, update_time, owner_pid, unique_id, category, scores, unk1, unk2) VALUES `
	args := []any{pid, uniqueID}
	n := 3
	for _, score := range scores {
		query += fmt.Sprintf(`(now(), now(), $1, $2, $%d, $%d, $%d, $%d)`, n, n+1, n+2, n+3)
		args = append(args, score.Category, pqextended.Array(score.Score), score.Unknown1, score.Unknown2)
		n += 4
	}
	query += ` ON CONFLICT (owner_pid, unique_id, category) DO UPDATE SET
 		deleted = false,
 		update_time = now(),
 		scores = EXCLUDED.scores, 
 		unk1 = EXCLUDED.unk1,
 		unk2 = EXCLUDED.unk2`

	commonglobals.Logger.Infof("Wow! %v\n%v", query, args)

	result, err := manager.Database.Exec(query, args...)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	}
	rows, err := result.RowsAffected()
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Core.SystemError, err.Error())
	} else if rows != int64(len(scores)) {
		commonglobals.Logger.Warningf("Only updated %v/%v scores", rows, len(scores))
	}

	return nil
}
