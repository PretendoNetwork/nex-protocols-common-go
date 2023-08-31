package common_globals

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	"golang.org/x/exp/slices"
)

// GetAvailableGatheringID returns a gathering ID which doesn't belong to any session
// Returns 0 if no IDs are available (math.MaxUint32 has been reached)
func GetAvailableGatheringID() uint32 {
	var gatheringID uint32 = 1
	for gatheringID < math.MaxUint32 {
		// * If the session does not exist, the gathering ID is free
		if _, ok := Sessions[gatheringID]; !ok {
			return gatheringID
		}

		gatheringID++
	}

	return 0
}

// FindOtherConnectionID searches a connection ID on the session that isn't the given one
// Returns 0 if no connection ID could be found
func FindOtherConnectionID(excludedConnectionID uint32, gatheringID uint32) uint32 {
	for _, connectionID := range Sessions[gatheringID].ConnectionIDs {
		if connectionID != excludedConnectionID {
			return connectionID
		}
	}

	return 0
}

// RemoveConnectionIDFromSession removes a client from the session
func RemoveConnectionIDFromSession(clientConnectionID uint32, gathering uint32) {
	for index, connectionID := range Sessions[gathering].ConnectionIDs {
		if connectionID == clientConnectionID {
			Sessions[gathering].ConnectionIDs = DeleteIndex(Sessions[gathering].ConnectionIDs, index)
		}
	}

	if len(Sessions[gathering].ConnectionIDs) == 0 {
		delete(Sessions, gathering)
	}
}

// FindClientSession searches for session the given connection ID is connected to
func FindClientSession(connectionID uint32) uint32 {
	for gatheringID := range Sessions {
		if slices.Contains(Sessions[gatheringID].ConnectionIDs, connectionID) {
			return gatheringID
		}
	}

	return 0
}

// RemoveConnectionIDFromAllSessions removes a client from every session
func RemoveConnectionIDFromAllSessions(clientConnectionID uint32) {
	// * Keep checking until no session is found
	for gid := FindClientSession(clientConnectionID); gid != 0; {
		RemoveConnectionIDFromSession(clientConnectionID, gid)

		gid = FindClientSession(clientConnectionID)
	}
}

// FindSessionByMatchmakeSession finds a gathering that matches with a MatchmakeSession
func FindSessionByMatchmakeSession(searchMatchmakeSession *match_making_types.MatchmakeSession) uint32 {
	// * This portion finds any sessions that match the search session
	// * It does not care about anything beyond that, such as if the match is already full
	// * This is handled below
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))
	for index, session := range Sessions {
		if session.SearchMatchmakeSession.Equals(searchMatchmakeSession) {
			candidateSessionIndexes = append(candidateSessionIndexes, index)
		}
	}

	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants) {
			continue
		}

		if !sessionToCheck.GameMatchmakeSession.OpenParticipation {
			continue
		}

		return sessionIndex // * Found a match
	}

	return 0
}

// FindSessionByMatchmakeSessionSearchCriterias finds a gathering that matches with a MatchmakeSession
func FindSessionByMatchmakeSessionSearchCriterias(lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria, gameSpecificChecks func(requestSearchCriteria, sessionSearchCriteria *match_making_types.MatchmakeSessionSearchCriteria) bool) uint32 {
	// * This portion finds any sessions that match the search session
	// * It does not care about anything beyond that, such as if the match is already full
	// * This is handled below.
	candidateSessionIndexes := make([]uint32, 0, len(Sessions))

	for index, session := range Sessions {
		if len(lstSearchCriteria) == len(session.SearchCriteria) {
			for criteriaIndex, sessionSearchCriteria := range session.SearchCriteria {
				requestSearchCriteria := lstSearchCriteria[criteriaIndex]

				// * Check things like game specific attributes
				if gameSpecificChecks != nil && !gameSpecificChecks(lstSearchCriteria[criteriaIndex], sessionSearchCriteria) {
					continue
				}

				if requestSearchCriteria.GameMode != "" && requestSearchCriteria.GameMode != sessionSearchCriteria.GameMode {
					continue
				}

				if requestSearchCriteria.MinParticipants != "" {
					split := strings.Split(requestSearchCriteria.MinParticipants, ",")
					minStr, maxStr := split[0], split[1]

					if minStr != "" {
						min, err := strconv.Atoi(minStr)
						if err != nil {
							// TODO - We don't have a logger here
							continue
						}

						if session.SearchMatchmakeSession.MinimumParticipants < uint16(min) {
							continue
						}
					}

					if maxStr != "" {
						max, err := strconv.Atoi(maxStr)
						if err != nil {
							// TODO - We don't have a logger here
							continue
						}

						if session.SearchMatchmakeSession.MinimumParticipants > uint16(max) {
							continue
						}
					}
				}

				if requestSearchCriteria.MaxParticipants != "" {
					split := strings.Split(requestSearchCriteria.MaxParticipants, ",")
					minStr := split[0]
					maxStr := ""

					if len(split) > 1 {
						maxStr = split[1]
					}

					if minStr != "" {
						min, err := strconv.Atoi(minStr)
						if err != nil {
							// TODO - We don't have a logger here
							continue
						}

						if session.SearchMatchmakeSession.MaximumParticipants < uint16(min) {
							continue
						}
					}

					if maxStr != "" {
						max, err := strconv.Atoi(maxStr)
						if err != nil {
							// TODO - We don't have a logger here
							continue
						}

						if session.SearchMatchmakeSession.MaximumParticipants > uint16(max) {
							continue
						}
					}
				}

				candidateSessionIndexes = append(candidateSessionIndexes, index)
			}
		}
	}

	// * Further filter the candidate sessions
	for _, sessionIndex := range candidateSessionIndexes {
		sessionToCheck := Sessions[sessionIndex]
		if len(sessionToCheck.ConnectionIDs) >= int(sessionToCheck.GameMatchmakeSession.MaximumParticipants) {
			continue
		}

		if !sessionToCheck.GameMatchmakeSession.OpenParticipation {
			continue
		}

		return sessionIndex // * Found a match
	}

	return 0
}

// AddPlayersToSession updates the given sessions state to include the provided connection IDs
// Returns a NEX error code if failed
func AddPlayersToSession(session *CommonMatchmakeSession, connectionIDs []uint32) (error, uint32) {
	if (len(session.ConnectionIDs) + len(connectionIDs)) > int(session.GameMatchmakeSession.Gathering.MaximumParticipants) {
		return fmt.Errorf("Gathering %d is full", session.GameMatchmakeSession.Gathering.ID), nex.Errors.RendezVous.SessionFull
	}

	for _, connectedID := range connectionIDs {
		if slices.Contains(session.ConnectionIDs, connectedID) {
			return fmt.Errorf("Connection ID %d is already in gathering %d", connectedID, session.GameMatchmakeSession.Gathering.ID), nex.Errors.RendezVous.AlreadyParticipatedGathering
		}

		session.ConnectionIDs = append(session.ConnectionIDs, connectedID)

		session.GameMatchmakeSession.ParticipationCount += 1
	}

	return nil, 0
}
