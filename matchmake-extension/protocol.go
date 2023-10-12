package matchmake_extension

import (
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	matchmake_extension_mario_kart_8 "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension/mario-kart-8"
	"github.com/PretendoNetwork/plogger-go"
)

var commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol
var logger = plogger.NewLogger()

type CommonMatchmakeExtensionProtocol struct {
	server             *nex.Server
	DefaultProtocol    *matchmake_extension.Protocol
	MarioKart8Protocol *matchmake_extension_mario_kart_8.Protocol

	cleanupSearchMatchmakeSessionHandler                    func(matchmakeSession *match_making_types.MatchmakeSession)
	cleanupMatchmakeSessionSearchCriteriaHandler            func(lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria)
	gameSpecificMatchmakeSessionSearchCriteriaChecksHandler func(requestSearchCriteria, sessionSearchCriteria *match_making_types.MatchmakeSessionSearchCriteria) bool
}

// CleanupSearchMatchmakeSession sets the CleanupSearchMatchmakeSession handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) CleanupSearchMatchmakeSession(handler func(matchmakeSession *match_making_types.MatchmakeSession)) {
	commonMatchmakeExtensionProtocol.cleanupSearchMatchmakeSessionHandler = handler
}

// CleanupMatchmakeSessionSearchCriteria sets the CleanupMatchmakeSessionSearchCriteria handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) CleanupMatchmakeSessionSearchCriteria(handler func(lstSearchCriteria []*match_making_types.MatchmakeSessionSearchCriteria)) {
	commonMatchmakeExtensionProtocol.cleanupMatchmakeSessionSearchCriteriaHandler = handler
}

// GameSpecificMatchmakeSessionSearchCriteriaChecks sets the GameSpecificMatchmakeSessionSearchCriteriaChecks handler function
func (commonMatchmakeExtensionProtocol *CommonMatchmakeExtensionProtocol) GameSpecificMatchmakeSessionSearchCriteriaChecks(handler func(requestSearchCriteria, sessionSearchCriteria *match_making_types.MatchmakeSessionSearchCriteria) bool) {
	commonMatchmakeExtensionProtocol.gameSpecificMatchmakeSessionSearchCriteriaChecksHandler = handler
}

func initDefault(c *CommonMatchmakeExtensionProtocol) {
	// TODO - Organize by method ID
	c.DefaultProtocol = matchmake_extension.NewProtocol(c.server)
	c.DefaultProtocol.OpenParticipation(openParticipation)
	c.DefaultProtocol.CloseParticipation(closeParticipation)
	c.DefaultProtocol.CreateMatchmakeSession(createMatchmakeSession)
	c.DefaultProtocol.GetSimplePlayingSession(getSimplePlayingSession)
	c.DefaultProtocol.AutoMatchmakePostpone(autoMatchmake_Postpone)
	c.DefaultProtocol.AutoMatchmakeWithParamPostpone(autoMatchmakeWithParam_Postpone)
	c.DefaultProtocol.AutoMatchmakeWithSearchCriteriaPostpone(autoMatchmakeWithSearchCriteria_Postpone)
	c.DefaultProtocol.UpdateProgressScore(updateProgressScore)
	c.DefaultProtocol.CreateMatchmakeSessionWithParam(createMatchmakeSessionWithParam)
	c.DefaultProtocol.UpdateApplicationBuffer(updateApplicationBuffer)
	c.DefaultProtocol.JoinMatchmakeSession(joinMatchmakeSession)
	c.DefaultProtocol.JoinMatchmakeSessionWithParam(joinMatchmakeSessionWithParam)
	c.DefaultProtocol.ModifyCurrentGameAttribute(modifyCurrentGameAttribute)
	c.DefaultProtocol.BrowseMatchmakeSession(browseMatchmakeSession)
}

func initMarioKart8(c *CommonMatchmakeExtensionProtocol) {
	// TODO - Organize by method ID
	c.MarioKart8Protocol = matchmake_extension_mario_kart_8.NewProtocol(c.server)
	c.MarioKart8Protocol.OpenParticipation(openParticipation)
	c.MarioKart8Protocol.CloseParticipation(closeParticipation)
	c.MarioKart8Protocol.CreateMatchmakeSession(createMatchmakeSession)
	c.MarioKart8Protocol.GetSimplePlayingSession(getSimplePlayingSession)
	c.MarioKart8Protocol.AutoMatchmakePostpone(autoMatchmake_Postpone)
	c.MarioKart8Protocol.AutoMatchmakeWithSearchCriteriaPostpone(autoMatchmakeWithSearchCriteria_Postpone)
	c.MarioKart8Protocol.UpdateProgressScore(updateProgressScore)
}

// NewCommonMatchmakeExtensionProtocol returns a new CommonMatchmakeExtensionProtocol
func NewCommonMatchmakeExtensionProtocol(server *nex.Server) *CommonMatchmakeExtensionProtocol {
	commonMatchmakeExtensionProtocol = &CommonMatchmakeExtensionProtocol{server: server}

	patch := server.MatchMakingProtocolVersion().GameSpecificPatch

	if strings.EqualFold(patch, "AMKJ") {
		logger.Info("Using Mario Kart 8 MatchmakeExtension protocol")
		initMarioKart8(commonMatchmakeExtensionProtocol)
	} else {
		if patch != "" {
			logger.Infof("Matchmaking version patch %q not recognized", patch)
		}

		logger.Info("Using default MatchmakeExtension protocol")
		initDefault(commonMatchmakeExtensionProtocol)
	}

	return commonMatchmakeExtensionProtocol
}
