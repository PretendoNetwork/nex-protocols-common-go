package matchmake_referee_database

import (
	"database/sql"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"
)

func StartRound(manager *common_globals.UtilityManager, param matchmake_referee_types.MatchmakeRefereeStartRoundParam) (types.UInt64, *nex.Error) {
	var participants types.List[types.PID]
	var roundId types.UInt64
	pidList := param.PIDs

	// * Query for participants from the Gathering ID to insert into the new round's row. We'll perform the check for the gathering not existing here.

	err := manager.Database.QueryRow(`SELECT participants FROM matchmaking.gatherings WHERE id = $1`, param.GID).Scan(&participants)
	if err == sql.ErrNoRows {
		common_globals.Logger.Error(err.Error())
		return roundId, nex.NewError(nex.ResultCodes.MatchmakeReferee.NotParticipatedGathering, "Gathering does not exist")
	} else if err != nil {
		common_globals.Logger.Error(err.Error())
		return roundId, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	// * Create the round's row in Postgres if it does not already exist. Thankfully, this only gets sent by one client so need to worry about duplicates

	// TODO - Are the states supposed to have a specific value, or is it server-assigned?

	err = manager.Database.QueryRow(`
    INSERT INTO matchmake_referee.rounds (
    gid, state, personal_data_category, participants) VALUES ($1, 0, $2, $3)
    RETURNING round_id`, param.GID, param.PersonalDataCategory, param.PIDs).Scan(&roundId)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return roundId, nex.NewError(nex.ResultCodes.Core.Unknown, "change_error")
	}

	// * Sort both lists of PIDs in ascending order so that they are compatible

	slices.Sort(participants)
	slices.Sort(pidList)

	// * Compare the lists

	bool := slices.Equal(participants, pidList)
	if bool == false {
		return roundId, nex.NewError(nex.ResultCodes.MatchmakeReferee.NotParticipatedGathering, "change_error")
	}

	// * If we've made it to this line of code, all checks are successful.

	return roundId, nil
}
