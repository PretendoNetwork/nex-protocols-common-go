package database

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	pqextended "github.com/PretendoNetwork/pq-extended"
)

// FindMatchmakeSessionBySearchCriteria finds matchmake sessions with the given search criterias
func FindMatchmakeSessionBySearchCriteria(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, searchCriterias []match_making_types.MatchmakeSessionSearchCriteria, resultRange types.ResultRange, sourceMatchmakeSession *match_making_types.MatchmakeSession) ([]match_making_types.MatchmakeSession, *nex.Error) {
	resultMatchmakeSessions := make([]match_making_types.MatchmakeSession, 0)

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	var friendList []uint32
	if manager.GetUserFriendPIDs != nil {
		friendList = manager.GetUserFriendPIDs(uint32(connection.PID()))
	}

	if resultRange.Offset == math.MaxUint32 {
		resultRange.Offset = 0
	}

	for _, searchCriteria := range searchCriterias {
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
			ms.refer_gid=$1 AND
			ms.codeword=$2 AND
			array_length(ms.attribs, 1)=$3 AND
			(CASE WHEN g.participation_policy=98 THEN g.owner_pid=ANY($4) ELSE true END) AND
			(CASE WHEN $5=true THEN ms.open_participation=true ELSE true END) AND
			(CASE WHEN $6=true THEN g.host_pid <> 0 ELSE true END) AND
			(CASE WHEN $7=true THEN ms.user_password_enabled=false ELSE true END) AND
			(CASE WHEN $8=true THEN ms.system_password_enabled=false ELSE true END)`

		var valid bool = true
		for i, attrib := range searchCriteria.Attribs {
			// * Ignore attribute 1 here, reserved for the selection method
			if i == 1 {
				continue;
			}

			if attrib != "" {
				before, after, found := strings.Cut(string(attrib), ",")
				if found {
					min, err := strconv.ParseUint(before, 10, 32)
					if err != nil {
						valid = false
						break
					}

					max, err := strconv.ParseUint(after, 10, 32)
					if err != nil {
						valid = false
						break
					}

					searchStatement += fmt.Sprintf(` AND ms.attribs[%d] BETWEEN %d AND %d`, i + 1, min, max)
				} else {
					value, err := strconv.ParseUint(before, 10, 32)
					if err != nil {
						valid = false
						break
					}

					searchStatement += fmt.Sprintf(` AND ms.attribs[%d]=%d`, i + 1, value)
				}
			}
		}

		// * Search criteria is invalid, continue to next one
		if !valid {
			continue
		}

		if searchCriteria.MaxParticipants != "" {
			before, after, found := strings.Cut(string(searchCriteria.MaxParticipants), ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND g.max_participants BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND g.max_participants=%d`, value)
			}
		}

		if searchCriteria.MinParticipants != "" {
			before, after, found := strings.Cut(string(searchCriteria.MinParticipants), ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND g.min_participants BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND g.min_participants=%d`, value)
			}
		}

		if searchCriteria.GameMode != "" {
			before, after, found := strings.Cut(string(searchCriteria.GameMode), ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 32)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 32)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.game_mode BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 32)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.game_mode=%d`, value)
			}
		}

		if searchCriteria.MatchmakeSystemType != "" {
			before, after, found := strings.Cut(string(searchCriteria.MatchmakeSystemType), ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 32)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 32)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.matchmake_system_type BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 32)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.matchmake_system_type=%d`, value)
			}
		}

		// * Filter full sessions if necessary
		if searchCriteria.VacantOnly {
			// * Account for the VacantParticipants when searching for sessions (if given)
			if searchCriteria.VacantParticipants == 0 {
				searchStatement += ` AND array_length(g.participants, 1) + 1 <= g.max_participants`
			} else {
				searchStatement += fmt.Sprintf(` AND array_length(g.participants, 1) + %d <= g.max_participants`, searchCriteria.VacantParticipants)
			}
		}

		switch constants.SelectionMethod(searchCriteria.SelectionMethod) {
		case constants.SelectionMethodRandom:
			// * Random global
			searchStatement += ` ORDER BY RANDOM()`
		case constants.SelectionMethodNearestNeighbor:
			// * Closest attribute
			attribute1, err := strconv.ParseUint(string(searchCriteria.Attribs[1]), 10, 32)
			if err != nil {
				globals.Logger.Error(err.Error())
				continue
			}

			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.attribs[2])`, attribute1)
		case constants.SelectionMethodBroadenRange:
			// * Ranked

			// TODO - Actually implement ranked matchmaking, using closest attribute at the moment
			attribute1, err := strconv.ParseUint(string(searchCriteria.Attribs[1]), 10, 32)
			if err != nil {
				globals.Logger.Error(err.Error())
				continue
			}

			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.attribs[2])`, attribute1)
		case constants.SelectionMethodProgressScore:
			// * Progress Score

			// * We can only use this when doing auto-matchmake
			if sourceMatchmakeSession == nil {
				continue
			}

			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.progress_score)`, sourceMatchmakeSession.ProgressScore)
		case constants.SelectionMethodBroadenRangeWithProgressScore:
			// * Ranked + Progress

			// TODO - Actually implement ranked matchmaking, using closest attribute at the moment

			// * We can only use this when doing auto-matchmake
			if sourceMatchmakeSession == nil {
				continue
			}

			attribute1, err := strconv.ParseUint(string(searchCriteria.Attribs[1]), 10, 32)
			if err != nil {
				globals.Logger.Error(err.Error())
				continue
			}

			// TODO - Should the attribute and the progress score actually weigh the same?
			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.attribs[2] + %d - ms.progress_score)`, attribute1, sourceMatchmakeSession.ProgressScore)
		// case constants.SelectionMethodScoreBased: // * According to notes this is related with the MatchmakeParam. TODO - Implement this
		}

		// * If the ResultRange inside the MatchmakeSessionSearchCriteria is valid (only present on NEX 4.0+), use that
		// * Otherwise, use the one given as argument
		if searchCriteria.ResultRange.Length != 0 {
			searchStatement += fmt.Sprintf(` LIMIT %d OFFSET %d`, uint32(searchCriteria.ResultRange.Length), uint32(searchCriteria.ResultRange.Offset))
		} else {
			// * Since we use one ResultRange for all searches, limit the total length to the one specified
			// * but apply the same offset to all queries
			searchStatement += fmt.Sprintf(` LIMIT %d OFFSET %d`, uint32(resultRange.Length) - uint32(len(resultMatchmakeSessions)), uint32(resultRange.Offset))
		}

		rows, err := manager.Database.Query(searchStatement,
			uint32(searchCriteria.ReferGID),
			string(searchCriteria.CodeWord),
			len(searchCriteria.Attribs),
			pqextended.Array(friendList),
			bool(searchCriteria.ExcludeLocked),
			bool(searchCriteria.ExcludeNonHostPID),
			bool(searchCriteria.ExcludeUserPasswordSet),
			bool(searchCriteria.ExcludeSystemPasswordSet),
		)
		if err != nil {
			globals.Logger.Critical(err.Error())
			continue
		}

		for rows.Next() {
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

			err = rows.Scan(
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
				globals.Logger.Critical(err.Error())
				continue
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

			resultMatchmakeSessions = append(resultMatchmakeSessions, resultMatchmakeSession)
		}

		rows.Close()
	}

	return resultMatchmakeSessions, nil
}
