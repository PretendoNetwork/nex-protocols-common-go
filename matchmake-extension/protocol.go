package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

type CommonProtocol struct {
	endpoint                                         nex.EndpointInterface
	protocol                                         matchmake_extension.Interface
	CleanupSearchMatchmakeSession                    func(matchmakeSession *match_making_types.MatchmakeSession)
	GameSpecificMatchmakeSessionSearchCriteriaChecks func(searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, matchmakeSession *match_making_types.MatchmakeSession) bool
}

// GetUserFriendPIDs sets the GetUserFriendPIDs handler function
func (commonProtocol *CommonProtocol) GetUserFriendPIDs(handler func(pid uint32) []uint32) {
	common_globals.GetUserFriendPIDsHandler = handler
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol matchmake_extension.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerOpenParticipation(commonProtocol.openParticipation)
	protocol.SetHandlerCloseParticipation(commonProtocol.closeParticipation)
	protocol.SetHandlerCreateMatchmakeSession(commonProtocol.createMatchmakeSession)
	protocol.SetHandlerGetSimplePlayingSession(commonProtocol.getSimplePlayingSession)
	protocol.SetHandlerAutoMatchmakePostpone(commonProtocol.autoMatchmake_Postpone)
	protocol.SetHandlerAutoMatchmakeWithParamPostpone(commonProtocol.autoMatchmakeWithParam_Postpone)
	protocol.SetHandlerAutoMatchmakeWithSearchCriteriaPostpone(commonProtocol.autoMatchmakeWithSearchCriteria_Postpone)
	protocol.SetHandlerUpdateProgressScore(commonProtocol.updateProgressScore)
	protocol.SetHandlerCreateMatchmakeSessionWithParam(commonProtocol.createMatchmakeSessionWithParam)
	protocol.SetHandlerUpdateApplicationBuffer(commonProtocol.updateApplicationBuffer)
	protocol.SetHandlerJoinMatchmakeSession(commonProtocol.joinMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionWithParam(commonProtocol.joinMatchmakeSessionWithParam)
	protocol.SetHandlerModifyCurrentGameAttribute(commonProtocol.modifyCurrentGameAttribute)
	protocol.SetHandlerBrowseMatchmakeSession(commonProtocol.browseMatchmakeSession)

	return commonProtocol
}
