package database

import (
	"database/sql"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// FindMatchmakeSession finds a matchmake session with the given search matchmake session
func FindMatchmakeSession(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, searchMatchmakeSession match_making_types.MatchmakeSession) (*match_making_types.MatchmakeSession, *nex.Error) {
	attribs := make([]uint32, len(searchMatchmakeSession.Attributes))
	for i, value := range searchMatchmakeSession.Attributes {
		attribs[i] = uint32(value)
	}

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	searchStatement := `SELECT
		g.id,
		g.owner_pid,
		g.host_pid,
		g.min_participants,
		g.max_participants,
		g.participation_policy,
		g.policy_argument,
		g.flags,
		g.state,
		g.description,
		array_length(g.participants, 1),
		g.started_time,
		ms.game_mode,
		ms.attribs,
		ms.open_participation,
		ms.matchmake_system_type,
		ms.application_buffer,
		ms.progress_score,
		ms.session_key,
		ms.option_zero,
		ms.matchmake_param,
		ms.user_password,
		ms.refer_gid,
		ms.user_password_enabled,
		ms.system_password_enabled,
		ms.codeword
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		g.host_pid <> 0 AND
		ms.open_participation=true AND
		array_length(g.participants, 1) < g.max_participants AND
		ms.user_password_enabled=false AND
		ms.system_password_enabled=false AND
		g.max_participants=$1 AND
		g.min_participants=$2 AND
		ms.game_mode=$3 AND
		ms.attribs[1]=$4 AND
		ms.attribs[3]=$6 AND
		ms.attribs[4]=$7 AND
		ms.attribs[5]=$8 AND
		ms.attribs[6]=$9 AND
		ms.matchmake_system_type=$10 AND
		ms.codeword=$11 AND (CASE WHEN g.participation_policy=98 THEN g.owner_pid=ANY($12) ELSE true END)
		ORDER BY abs($5 - ms.attribs[2])` // * Use "Closest attribute" selection method, guessing from Mario Kart 7

	var friendList []uint32
	// * Prevent access to friend rooms if not implemented
	if manager.GetUserFriendPIDs != nil {
		friendList = manager.GetUserFriendPIDs(uint32(connection.PID()))
	}

	resultMatchmakeSession := match_making_types.NewMatchmakeSession()
	var startedTime time.Time
	var resultAttribs []uint32
	var resultMatchmakeParam []byte

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := manager.Database.QueryRow(searchStatement,
		searchMatchmakeSession.Gathering.MaximumParticipants,
		searchMatchmakeSession.Gathering.MinimumParticipants,
		searchMatchmakeSession.GameMode,
		attribs[0],
		attribs[1],
		attribs[2],
		attribs[3],
		attribs[4],
		attribs[5],
		searchMatchmakeSession.MatchmakeSystemType,
		searchMatchmakeSession.CodeWord,
		pqextended.Array(friendList),
	).Scan(
		&resultMatchmakeSession.Gathering.ID,
		&resultMatchmakeSession.Gathering.OwnerPID,
		&resultMatchmakeSession.Gathering.HostPID,
		&resultMatchmakeSession.Gathering.MinimumParticipants,
		&resultMatchmakeSession.Gathering.MaximumParticipants,
		&resultMatchmakeSession.Gathering.ParticipationPolicy,
		&resultMatchmakeSession.Gathering.PolicyArgument,
		&resultMatchmakeSession.Gathering.Flags,
		&resultMatchmakeSession.Gathering.State,
		&resultMatchmakeSession.Gathering.Description,
		&resultMatchmakeSession.ParticipationCount,
		&startedTime,
		&resultMatchmakeSession.GameMode,
		pqextended.Array(&resultAttribs),
		&resultMatchmakeSession.OpenParticipation,
		&resultMatchmakeSession.MatchmakeSystemType,
		&resultMatchmakeSession.ApplicationBuffer,
		&resultMatchmakeSession.ProgressScore,
		&resultMatchmakeSession.SessionKey,
		&resultMatchmakeSession.Option,
		&resultMatchmakeParam,
		&resultMatchmakeSession.UserPassword,
		&resultMatchmakeSession.ReferGID,
		&resultMatchmakeSession.UserPasswordEnabled,
		&resultMatchmakeSession.SystemPasswordEnabled,
		&resultMatchmakeSession.CodeWord,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultMatchmakeSession.Attributes = attributesSlice

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	return &resultMatchmakeSession, nil
}
