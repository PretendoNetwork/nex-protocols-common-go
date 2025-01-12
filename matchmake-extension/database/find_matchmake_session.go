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

	var gatheringID uint32
	var ownerPID uint64
	var hostPID uint64
	var minimumParticipants uint16
	var maximumParticipants uint16
	var participationPolicy uint32
	var policyArgument uint32
	var flags uint32
	var state uint32
	var description string
	var participationCount uint32
	var startedTime time.Time
	var gameMode uint32
	var resultAttribs []uint32
	var openParticipation bool
	var matchmakeSystemType uint32
	var applicationBuffer []byte
	var progressScore uint8
	var sessionKey []byte
	var option uint32
	var resultMatchmakeParam []byte
	var userPassword string
	var referGID uint32
	var userPasswordEnabled bool
	var systemPasswordEnabled bool
	var codeWord string

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := manager.Database.QueryRow(searchStatement,
		uint16(searchMatchmakeSession.Gathering.MaximumParticipants),
		uint16(searchMatchmakeSession.Gathering.MinimumParticipants),
		uint32(searchMatchmakeSession.GameMode),
		attribs[0],
		attribs[1],
		attribs[2],
		attribs[3],
		attribs[4],
		attribs[5],
		uint32(searchMatchmakeSession.MatchmakeSystemType),
		string(searchMatchmakeSession.CodeWord),
		pqextended.Array(friendList),
	).Scan(
		&gatheringID,
		&ownerPID,
		&hostPID,
		&minimumParticipants,
		&maximumParticipants,
		&participationPolicy,
		&policyArgument,
		&flags,
		&state,
		&description,
		&participationCount,
		&startedTime,
		&gameMode,
		pqextended.Array(&resultAttribs),
		&openParticipation,
		&matchmakeSystemType,
		&applicationBuffer,
		&progressScore,
		&sessionKey,
		&option,
		&resultMatchmakeParam,
		&userPassword,
		&referGID,
		&userPasswordEnabled,
		&systemPasswordEnabled,
		&codeWord,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} else {
			return nil, nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
		}
	}

	resultMatchmakeSession := match_making_types.NewMatchmakeSession()

	resultMatchmakeSession.Gathering.ID = types.NewUInt32(gatheringID)
	resultMatchmakeSession.OwnerPID = types.NewPID(ownerPID)
	resultMatchmakeSession.HostPID = types.NewPID(hostPID)
	resultMatchmakeSession.Gathering.MinimumParticipants = types.NewUInt16(minimumParticipants)
	resultMatchmakeSession.Gathering.MaximumParticipants = types.NewUInt16(maximumParticipants)
	resultMatchmakeSession.Gathering.ParticipationPolicy = types.NewUInt32(participationPolicy)
	resultMatchmakeSession.Gathering.PolicyArgument = types.NewUInt32(policyArgument)
	resultMatchmakeSession.Gathering.Flags = types.NewUInt32(flags)
	resultMatchmakeSession.Gathering.State = types.NewUInt32(state)
	resultMatchmakeSession.Gathering.Description = types.NewString(description)
	resultMatchmakeSession.ParticipationCount = types.NewUInt32(participationCount)
	resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)
	resultMatchmakeSession.GameMode = types.NewUInt32(gameMode)

	attributesSlice := make([]types.UInt32, len(resultAttribs))
	for i, value := range resultAttribs {
		attributesSlice[i] = types.NewUInt32(value)
	}
	resultMatchmakeSession.Attributes = attributesSlice

	resultMatchmakeSession.OpenParticipation = types.NewBool(openParticipation)
	resultMatchmakeSession.MatchmakeSystemType = types.NewUInt32(matchmakeSystemType)
	resultMatchmakeSession.ApplicationBuffer = applicationBuffer
	resultMatchmakeSession.ProgressScore = types.NewUInt8(progressScore)
	resultMatchmakeSession.SessionKey = sessionKey
	resultMatchmakeSession.Option = types.UInt32(option)

	matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
	resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

	resultMatchmakeSession.UserPassword = types.NewString(userPassword)
	resultMatchmakeSession.ReferGID = types.NewUInt32(referGID)
	resultMatchmakeSession.UserPasswordEnabled = types.NewBool(userPasswordEnabled)
	resultMatchmakeSession.SystemPasswordEnabled = types.NewBool(systemPasswordEnabled)
	resultMatchmakeSession.CodeWord = types.String(codeWord)

	return &resultMatchmakeSession, nil
}
