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
func FindMatchmakeSessionBySearchCriteria(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, searchCriterias []*match_making_types.MatchmakeSessionSearchCriteria, resultRange *types.ResultRange, sourceMatchmakeSession *match_making_types.MatchmakeSession) ([]*match_making_types.MatchmakeSession, *nex.Error) {
	resultMatchmakeSessions := make([]*match_making_types.MatchmakeSession, 0)

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	var friendList []uint32
	if manager.GetUserFriendPIDs != nil {
		friendList = manager.GetUserFriendPIDs(connection.PID().LegacyValue())
	}

	if resultRange.Offset.Value == math.MaxUint32 {
		resultRange.Offset.Value = 0
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
			ms.open_participation=true AND
			ms.refer_gid=$1 AND
			ms.codeword=$2 AND
			array_length(ms.attribs, 1)=$3 AND
			(CASE WHEN g.participation_policy=98 THEN g.owner_pid=ANY($4) ELSE true END) AND
			(CASE WHEN $5=true THEN ms.open_participation=true ELSE true END) AND
			(CASE WHEN $6=true THEN g.host_pid <> 0 ELSE true END) AND
			(CASE WHEN $7=true THEN ms.user_password_enabled=false ELSE true END) AND
			(CASE WHEN $8=true THEN ms.system_password_enabled=false ELSE true END)`

		var valid bool = true
		for i, attrib := range searchCriteria.Attribs.Slice() {
			if attrib.Value != "" {
				before, after, found := strings.Cut(attrib.Value, ",")
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

		if searchCriteria.MaxParticipants.Value != "" {
			before, after, found := strings.Cut(searchCriteria.MaxParticipants.Value, ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.max_participants BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.max_participants=%d`, value)
			}
		}

		if searchCriteria.MinParticipants.Value != "" {
			before, after, found := strings.Cut(searchCriteria.MinParticipants.Value, ",")
			if found {
				min, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				max, err := strconv.ParseUint(after, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.min_participants BETWEEN %d AND %d`, min, max)
			} else {
				value, err := strconv.ParseUint(before, 10, 16)
				if err != nil {
					continue
				}

				searchStatement += fmt.Sprintf(` AND ms.min_participants=%d`, value)
			}
		}

		if searchCriteria.GameMode.Value != "" {
			before, after, found := strings.Cut(searchCriteria.GameMode.Value, ",")
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

		if searchCriteria.MatchmakeSystemType.Value != "" {
			before, after, found := strings.Cut(searchCriteria.MatchmakeSystemType.Value, ",")
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
		if searchCriteria.VacantOnly.Value {
			// * Account for the VacantParticipants when searching for sessions (if given)
			if searchCriteria.VacantParticipants.Value == 0 {
				searchStatement += ` AND array_length(g.participants, 1) + 1 <= g.max_participants`
			} else {
				searchStatement += fmt.Sprintf(` AND array_length(g.participants, 1) + %d <= g.max_participants`, searchCriteria.VacantParticipants.Value)
			}
		}

		switch constants.SelectionMethod(searchCriteria.SelectionMethod.Value) {
		case constants.SelectionMethodRandom:
			// * Random global
			searchStatement += ` ORDER BY RANDOM()`
		case constants.SelectionMethodNearestNeighbor:
			// * Closest attribute
			attribute1, err := strconv.ParseUint(searchCriteria.Attribs.Slice()[1].Value, 10, 32)
			if err != nil {
				globals.Logger.Error(err.Error())
				continue
			}

			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.attribs[2])`, attribute1)
		case constants.SelectionMethodBroadenRange:
			// * Ranked

			// TODO - Actually implement ranked matchmaking, using closest attribute at the moment
			attribute1, err := strconv.ParseUint(searchCriteria.Attribs.Slice()[1].Value, 10, 32)
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

			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.progress_score)`, sourceMatchmakeSession.ProgressScore.Value)
		case constants.SelectionMethodBroadenRangeWithProgressScore:
			// * Ranked + Progress

			// TODO - Actually implement ranked matchmaking, using closest attribute at the moment

			// * We can only use this when doing auto-matchmake
			if sourceMatchmakeSession == nil {
				continue
			}

			attribute1, err := strconv.ParseUint(searchCriteria.Attribs.Slice()[1].Value, 10, 32)
			if err != nil {
				globals.Logger.Error(err.Error())
				continue
			}

			// TODO - Should the attribute and the progress score actually weigh the same?
			searchStatement += fmt.Sprintf(` ORDER BY abs(%d - ms.attribs[2] + %d - ms.progress_score)`, attribute1, sourceMatchmakeSession.ProgressScore.Value)
		// case constants.SelectionMethodScoreBased: // * According to notes this is related with the MatchmakeParam. TODO - Implement this
		}

		// * If the ResultRange inside the MatchmakeSessionSearchCriteria is valid (only present on NEX 4.0+), use that
		// * Otherwise, use the one given as argument
		if searchCriteria.ResultRange.Length.Value != 0 {
			searchStatement += fmt.Sprintf(` LIMIT %d OFFSET %d`, searchCriteria.ResultRange.Length.Value, searchCriteria.ResultRange.Offset.Value)
		} else {
			// * Since we use one ResultRange for all searches, limit the total length to the one specified
			// * but apply the same offset to all queries
			searchStatement += fmt.Sprintf(` LIMIT %d OFFSET %d`, resultRange.Length.Value - uint32(len(resultMatchmakeSessions)), resultRange.Offset.Value)
		}

		rows, err := manager.Database.Query(searchStatement,
			searchCriteria.ReferGID.Value,
			searchCriteria.CodeWord.Value,
			searchCriteria.Attribs.Length(),
			pqextended.Array(friendList),
			searchCriteria.ExcludeLocked.Value,
			searchCriteria.ExcludeNonHostPID.Value,
			searchCriteria.ExcludeUserPasswordSet.Value,
			searchCriteria.ExcludeSystemPasswordSet.Value,
		)
		if err != nil {
			globals.Logger.Critical(err.Error())
			continue
		}

		for rows.Next() {
			resultMatchmakeSession := match_making_types.NewMatchmakeSession()
			var ownerPID uint64
			var hostPID uint64
			var startedTime time.Time
			var resultAttribs []uint32
			var resultMatchmakeParam []byte

			err = rows.Scan(
				&resultMatchmakeSession.Gathering.ID.Value,
				&ownerPID,
				&hostPID,
				&resultMatchmakeSession.Gathering.MinimumParticipants.Value,
				&resultMatchmakeSession.Gathering.MaximumParticipants.Value,
				&resultMatchmakeSession.Gathering.ParticipationPolicy.Value,
				&resultMatchmakeSession.Gathering.PolicyArgument.Value,
				&resultMatchmakeSession.Gathering.Flags.Value,
				&resultMatchmakeSession.Gathering.State.Value,
				&resultMatchmakeSession.Gathering.Description.Value,
				&resultMatchmakeSession.ParticipationCount.Value,
				&startedTime,
				&resultMatchmakeSession.GameMode.Value,
				pqextended.Array(&resultAttribs),
				&resultMatchmakeSession.OpenParticipation.Value,
				&resultMatchmakeSession.MatchmakeSystemType.Value,
				&resultMatchmakeSession.ApplicationBuffer.Value,
				&resultMatchmakeSession.ProgressScore.Value,
				&resultMatchmakeSession.SessionKey.Value,
				&resultMatchmakeSession.Option.Value,
				&resultMatchmakeParam,
				&resultMatchmakeSession.UserPassword.Value,
				&resultMatchmakeSession.ReferGID.Value,
				&resultMatchmakeSession.UserPasswordEnabled.Value,
				&resultMatchmakeSession.SystemPasswordEnabled.Value,
				&resultMatchmakeSession.CodeWord.Value,
			)
			if err != nil {
				globals.Logger.Critical(err.Error())
				continue
			}

			resultMatchmakeSession.OwnerPID = types.NewPID(ownerPID)
			resultMatchmakeSession.HostPID = types.NewPID(hostPID)
			resultMatchmakeSession.StartedTime = resultMatchmakeSession.StartedTime.FromTimestamp(startedTime)

			attributesSlice := make([]*types.PrimitiveU32, len(resultAttribs))
			for i, value := range resultAttribs {
				attributesSlice[i] = types.NewPrimitiveU32(value)
			}
			resultMatchmakeSession.Attributes.SetFromData(attributesSlice)

			matchmakeParamBytes := nex.NewByteStreamIn(resultMatchmakeParam, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())
			resultMatchmakeSession.MatchmakeParam.ExtractFrom(matchmakeParamBytes)

			resultMatchmakeSessions = append(resultMatchmakeSessions, resultMatchmakeSession)
		}

		rows.Close()
	}

	return resultMatchmakeSessions, nil
}
