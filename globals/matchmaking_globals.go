package common_globals

import (
	"database/sql"
	"sync"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	"github.com/PretendoNetwork/nex-protocols-go/v2/match-making/constants"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
)

// MatchmakingManager manages a matchmaking instance
type MatchmakingManager struct {
	Database                     *sql.DB
	Endpoint                     *nex.PRUDPEndPoint
	Mutex                        *sync.RWMutex
	MatchmakeSessionAttrNum      int
	MatchmakeSessionBufferLength int
	GetUserFriendPIDs            func(pid uint32) []uint32
	GetDetailedGatheringByID     func(manager *MatchmakingManager, sourcePID uint64, gatheringID uint32) (types.RVType, string, *nex.Error)
}

// CheckValidGathering checks if a Gathering is valid
func (mm *MatchmakingManager) CheckValidGathering(gathering match_making_types.Gathering) bool {
	if len(gathering.Description) > 256 {
		return false
	}

	return true
}

// CheckValidMatchmakeSession checks if a MatchmakeSession is valid
func (mm *MatchmakingManager) CheckValidMatchmakeSession(matchmakeSession match_making_types.MatchmakeSession) bool {
	if !mm.CheckValidGathering(matchmakeSession.Gathering) {
		return false
	}

	if len(matchmakeSession.Attributes) != mm.MatchmakeSessionAttrNum {
		return false
	}

	if matchmakeSession.ProgressScore > 100 {
		return false
	}

	if len(matchmakeSession.UserPassword) > 32 {
		return false
	}

	// * Except for UserPassword, all strings must have a length lower than 256
	if len(matchmakeSession.CodeWord) > 256 {
		return false
	}

	// * All buffers must have a length lower than 512
	if len(matchmakeSession.ApplicationBuffer) > mm.MatchmakeSessionBufferLength {
		return false
	}

	if len(matchmakeSession.SessionKey) > 512 {
		return false
	}

	return true
}

// CheckValidPersistentGathering checks if a PersistentGathering is valid
func (mm *MatchmakingManager) CheckValidPersistentGathering(persistentGathering match_making_types.PersistentGathering) bool {
	if !mm.CheckValidGathering(persistentGathering.Gathering) {
		return false
	}

	// * Only allow normal and password-protected community types
	if uint32(persistentGathering.CommunityType) != uint32(constants.PersistentGatheringTypeOpen) && uint32(persistentGathering.CommunityType) != uint32(constants.PersistentGatheringTypePasswordLocked) {
		return false
	}

	// * The UserPassword from a MatchmakeSession can be up to 32 characters, assuming the same here
	//
	// TODO - IS this actually the case?
	if len(persistentGathering.Password) > 32 {
		return false
	}

	if len(persistentGathering.Attribs) != 6 {
		return false
	}

	// * All buffers must have a length lower than 512
	if len(persistentGathering.ApplicationBuffer) > 512 {
		return false
	}

	// * Check that the participation dates are within bounds of the current date
	currentTime := types.NewDateTime(0).Now()

	if persistentGathering.ParticipationStartDate != 0 && persistentGathering.ParticipationStartDate > currentTime {
		return false
	}

	if persistentGathering.ParticipationEndDate != 0 && persistentGathering.ParticipationEndDate < currentTime {
		return false
	}

	return true
}

// NewMatchmakingManager returns a new MatchmakingManager
func NewMatchmakingManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *MatchmakingManager {
	return &MatchmakingManager{
		Endpoint:                     endpoint,
		Database:                     db,
		MatchmakeSessionAttrNum:      constants.NumMatchmakeSessionAttributes,
		MatchmakeSessionBufferLength: constants.MatchmakeBufferMaxLength,
		Mutex:                        &sync.RWMutex{},
	}
}
