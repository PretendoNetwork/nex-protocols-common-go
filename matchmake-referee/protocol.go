package matchmake_referee

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	matchmake_referee "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol matchmake_referee.Interface

	// * MatchmakeReferee and Utility seem to potentially be tied together due to Unique IDs. I'm going to go ahead and use a UtilityManager for this as a result.
	// * This may change in the future though.

	manager *common_globals.UtilityManager
}

func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.UtilityManager) {
	var err error

	commonProtocol.manager = manager

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS matchmake_referee`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// Create the stats table. This is based on the MatchmakeRefereeStats structure, while also including the PID of the owner

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmake_referee.stats (
        unique_id numeric(20), 
        category int,
        owner_pid numeric(20),

        recent_disconnection int,
        recent_violation int,
        recent_mismatch int,
        recent_win int,
        recent_loss int,
        recent_draw int,
        total_disconnect int,
        total_violation int,
        total_mismatch int,
        total_win int,
        total_loss int,
        total_draw int,
        rating_value bigint,

        PRIMARY KEY (unique_id, category, owner_pid)
        )`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// Previously made the assumption that round_id and gid would always be the same. This is definitely not the case as I look further into it.
	// round_id will remain as a bigserial, but gid will be shifted into being a bigint
	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmake_referee.rounds (
        round_id bigserial PRIMARY KEY,
        gid bigint,
        state int,
        personal_data_category int,
        participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[]
    )`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmake_referee.round_results (
		round_id bigint,
		pid numeric(20),
		personal_round_result_flag int,
		round_win_loss int,
		rating_value_change bigint,
		buffer bytea,

		PRIMARY KEY (round_id, pid)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(endpoint nex.EndpointInterface, protocol matchmake_referee.Interface) *CommonProtocol {

	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerStartRound(commonProtocol.StartRound)
	protocol.SetHandlerEndRound(commonProtocol.EndRound)
	protocol.SetHandlerEndRoundWithoutReport(commonProtocol.EndRoundWithoutReport)
	protocol.SetHandlerGetOrCreateStats(commonProtocol.GetOrCreateStats)

	return commonProtocol
}
