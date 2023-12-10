package matchmake_extension

import (
	nex "github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

var commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol

type CommonMatchmakeExtensionProtocol struct {
	server                                           nex.ServerInterface
	protocol                                         matchmake_extension.Interface
	CleanupSearchMatchmakeSession                    func(matchmakeSession *match_making_types.MatchmakeSession)
	GameSpecificMatchmakeSessionSearchCriteriaChecks func(searchCriteria *match_making_types.MatchmakeSessionSearchCriteria, matchmakeSession *match_making_types.MatchmakeSession) bool
}

// GetUserFriendPIDs sets the GetUserFriendPIDs handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) GetUserFriendPIDs(handler func(pid uint32) []uint32) {
	common_globals.GetUserFriendPIDsHandler = handler
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchmakeExtensionProtocol(protocol matchmake_extension.Interface) *CommonMatchmakeExtensionProtocol {
	protocol.SetHandlerOpenParticipation(openParticipation)
	protocol.SetHandlerCloseParticipation(closeParticipation)
	protocol.SetHandlerCreateMatchmakeSession(createMatchmakeSession)
	protocol.SetHandlerGetSimplePlayingSession(getSimplePlayingSession)
	protocol.SetHandlerAutoMatchmakePostpone(autoMatchmake_Postpone)
	protocol.SetHandlerAutoMatchmakeWithParamPostpone(autoMatchmakeWithParam_Postpone)
	protocol.SetHandlerAutoMatchmakeWithSearchCriteriaPostpone(autoMatchmakeWithSearchCriteria_Postpone)
	protocol.SetHandlerUpdateProgressScore(updateProgressScore)
	protocol.SetHandlerCreateMatchmakeSessionWithParam(createMatchmakeSessionWithParam)
	protocol.SetHandlerUpdateApplicationBuffer(updateApplicationBuffer)
	protocol.SetHandlerJoinMatchmakeSession(joinMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionWithParam(joinMatchmakeSessionWithParam)
	protocol.SetHandlerModifyCurrentGameAttribute(modifyCurrentGameAttribute)
	protocol.SetHandlerBrowseMatchmakeSession(browseMatchmakeSession)

	commonMatchmakeExtensionProtocol = &CommonMatchmakeExtensionProtocol{
		server:   protocol.Server(),
		protocol: protocol,
	}

	return commonMatchmakeExtensionProtocol
}
