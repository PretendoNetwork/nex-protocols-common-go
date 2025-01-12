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

// GetMatchmakeSessionByID gets a matchmake session with the given gathering ID and the system password
func GetMatchmakeSessionByID(manager *common_globals.MatchmakingManager, endpoint *nex.PRUDPEndPoint, gatheringID uint32) (match_making_types.MatchmakeSession, string, *nex.Error) {
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
	var systemPassword string

	// * For simplicity, we will only compare the values that exist on a MatchmakeSessionSearchCriteria
	err := manager.Database.QueryRow(`SELECT
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
		ms.codeword,
		ms.system_password
		FROM matchmaking.gatherings AS g
		INNER JOIN matchmaking.matchmake_sessions AS ms ON ms.id = g.id
		WHERE
		g.registered=true AND
		g.type='MatchmakeSession' AND
		g.id=$1`,
		gatheringID,
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
		&systemPassword,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return match_making_types.NewMatchmakeSession(), "", nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
		} else {
			return match_making_types.NewMatchmakeSession(), "", nex.NewError(nex.ResultCodes.Core.Unknown, err.Error())
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

	return resultMatchmakeSession, systemPassword, nil
}
