package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-extension"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-extension/database"
)

type CommonProtocol struct {
	endpoint                                       nex.EndpointInterface
	protocol                                       matchmake_extension.Interface
	manager                                        *common_globals.MatchmakingManager
	PersistentGatheringCreationMax                 int
	CanJoinMatchmakeSession                        func(manager *common_globals.MatchmakingManager, pid types.PID, matchmakeSession match_making_types.MatchmakeSession) *nex.Error
	CleanupSearchMatchmakeSession                  func(matchmakeSession *match_making_types.MatchmakeSession)
	CleanupMatchmakeSessionSearchCriterias         func(searchCriterias types.List[match_making_types.MatchmakeSessionSearchCriteria])
	OnAfterOpenParticipation                       func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterCloseParticipation                      func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterCreateMatchmakeSession                  func(packet nex.PacketInterface, anyGathering match_making_types.GatheringHolder, message types.String, participationCount types.UInt16)
	OnAfterGetSimplePlayingSession                 func(packet nex.PacketInterface, listPID types.List[types.PID], includeLoginUser types.Bool)
	OnAfterAutoMatchmakePostpone                   func(packet nex.PacketInterface, anyGathering match_making_types.GatheringHolder, message types.String)
	OnAfterAutoMatchmakeWithParamPostpone          func(packet nex.PacketInterface, autoMatchmakeParam match_making_types.AutoMatchmakeParam)
	OnAfterAutoMatchmakeWithSearchCriteriaPostpone func(packet nex.PacketInterface, lstSearchCriteria types.List[match_making_types.MatchmakeSessionSearchCriteria], anyGathering match_making_types.GatheringHolder, strMessage types.String)
	OnAfterGetPlayingSession                       func(packet nex.PacketInterface, lstPID types.List[types.PID])
	OnAfterCreateCommunity                         func(packet nex.PacketInterface, community match_making_types.PersistentGathering, strMessage types.String)
	OnAfterFindCommunityByGatheringID              func(packet nex.PacketInterface, lstGID types.List[types.UInt32])
	OnAfterFindOfficialCommunity                   func(packet nex.PacketInterface, isAvailableOnly types.Bool, resultRange types.ResultRange)
	OnAfterFindCommunityByParticipant              func(packet nex.PacketInterface, pid types.PID, resultRange types.ResultRange)
	OnAfterUpdateProgressScore                     func(packet nex.PacketInterface, gid types.UInt32, progressScore types.UInt8)
	OnAfterCreateMatchmakeSessionWithParam         func(packet nex.PacketInterface, createMatchmakeSessionParam match_making_types.CreateMatchmakeSessionParam)
	OnAfterUpdateApplicationBuffer                 func(packet nex.PacketInterface, gid types.UInt32, applicationBuffer types.Buffer)
	OnAfterJoinMatchmakeSession                    func(packet nex.PacketInterface, gid types.UInt32, strMessage types.String)
	OnAfterJoinMatchmakeSessionWithParam           func(packet nex.PacketInterface, joinMatchmakeSessionParam match_making_types.JoinMatchmakeSessionParam)
	OnAfterModifyCurrentGameAttribute              func(packet nex.PacketInterface, gid types.UInt32, attribIndex types.UInt32, newValue types.UInt32)
	OnAfterBrowseMatchmakeSession                  func(packet nex.PacketInterface, searchCriteria match_making_types.MatchmakeSessionSearchCriteria, resultRange types.ResultRange)
	OnAfterJoinMatchmakeSessionEx                  func(packet nex.PacketInterface, gid types.UInt32, strMessage types.String, dontCareMyBlockList types.Bool, participationCount types.UInt16)
	OnAfterGetSimpleCommunity                      func(packet nex.PacketInterface, gatheringIDList types.List[types.UInt32])
	OnAfterUpdateNotificationData                  func(packet nex.PacketInterface, uiType types.UInt32, uiParam1 types.UInt64, uiParam2 types.UInt64, strParam types.String)
	OnAfterGetFriendNotificationData               func(packet nex.PacketInterface, uiType types.Int32)
	OnAfterGetlstFriendNotificationData            func(packet nex.PacketInterface, lstTypes types.List[types.UInt32])
	OnAfterAddToBlockList                          func(packet nex.PacketInterface, lstPrincipalID types.List[types.PID])
	OnAfterRemoveFromBlockList                     func(packet nex.PacketInterface, lstPrincipalID types.List[types.PID])
	OnAfterGetMyBlockList                          func(packet nex.PacketInterface)
}

// SetDatabase defines the matchmaking manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.MatchmakingManager) {
	var err error

	commonProtocol.manager = manager

	manager.GetDetailedGatheringByID = database.GetDetailedGatheringByID

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.matchmake_sessions (
		id bigserial PRIMARY KEY,
		game_mode bigint,
		attribs bigint[],
		open_participation boolean,
		matchmake_system_type bigint,
		application_buffer bytea,
		flags bigint,
		state bigint,
		progress_score smallint,
		session_key bytea,
		option_zero bigint,
		matchmake_param bytea,
		user_password text,
		refer_gid bigint,
		user_password_enabled boolean,
		system_password_enabled boolean,
		codeword text,
		system_password text NOT NULL DEFAULT ''
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.persistent_gatherings (
		id bigserial PRIMARY KEY,
		community_type bigint,
		password text,
		attribs bigint[],
		application_buffer bytea,
		participation_start_date timestamp,
		participation_end_date timestamp
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.community_participations (
		id bigserial PRIMARY KEY,
		user_pid numeric(20),
		gathering_id bigint,
		participation_count bigint,
		UNIQUE (user_pid, gathering_id)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.notifications (
		id bigserial PRIMARY KEY,
		source_pid numeric(20),
		type bigint,
		param_1 numeric(20),
		param_2 numeric(20),
		param_str text,
		active boolean NOT NULL DEFAULT true,
		UNIQUE (source_pid, type)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.block_lists (
		user_pid numeric(20) NOT NULL,
		blocked_pid numeric(20) NOT NULL,
		PRIMARY KEY (user_pid, blocked_pid)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.participate_community (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		community_gid bigint,
		gathering_id bigint,
		participation_count bigint
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.notification_data (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		type bigint,
		param_1 bigint,
		param_2 bigint,
		param_str text
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * In case the server is restarted, unregister any previous matchmake sessions
	_, err = manager.Database.Exec(`UPDATE matchmaking.gatherings SET registered=false WHERE type='MatchmakeSession'`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	// * Mark all notifications as inactive
	_, err = manager.Database.Exec(`UPDATE matchmaking.notifications SET active=false`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol matchmake_extension.Interface) *CommonProtocol {
	endpoint := protocol.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol := &CommonProtocol{
		endpoint:                       endpoint,
		protocol:                       protocol,
		PersistentGatheringCreationMax: 4, // * Default of 4 active persistent gatherings per user
	}

	protocol.SetHandlerOpenParticipation(commonProtocol.openParticipation)
	protocol.SetHandlerCloseParticipation(commonProtocol.closeParticipation)
	protocol.SetHandlerCreateMatchmakeSession(commonProtocol.createMatchmakeSession)
	protocol.SetHandlerGetSimplePlayingSession(commonProtocol.getSimplePlayingSession)
	protocol.SetHandlerAutoMatchmakePostpone(commonProtocol.autoMatchmakePostpone)
	protocol.SetHandlerAutoMatchmakeWithParamPostpone(commonProtocol.autoMatchmakeWithParamPostpone)
	protocol.SetHandlerAutoMatchmakeWithSearchCriteriaPostpone(commonProtocol.autoMatchmakeWithSearchCriteriaPostpone)
	protocol.SetHandlerGetPlayingSession(commonProtocol.getPlayingSession)
	protocol.SetHandlerCreateCommunity(commonProtocol.createCommunity)
	protocol.SetHandlerFindCommunityByGatheringID(commonProtocol.findCommunityByGatheringID)
	protocol.SetHandlerFindOfficialCommunity(commonProtocol.findOfficialCommunity)
	protocol.SetHandlerFindCommunityByParticipant(commonProtocol.findCommunityByParticipant)
	protocol.SetHandlerUpdateProgressScore(commonProtocol.updateProgressScore)
	protocol.SetHandlerCreateMatchmakeSessionWithParam(commonProtocol.createMatchmakeSessionWithParam)
	protocol.SetHandlerUpdateApplicationBuffer(commonProtocol.updateApplicationBuffer)
	protocol.SetHandlerJoinMatchmakeSession(commonProtocol.joinMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionWithParam(commonProtocol.joinMatchmakeSessionWithParam)
	protocol.SetHandlerModifyCurrentGameAttribute(commonProtocol.modifyCurrentGameAttribute)
	protocol.SetHandlerBrowseMatchmakeSession(commonProtocol.browseMatchmakeSession)
	protocol.SetHandlerJoinMatchmakeSessionEx(commonProtocol.joinMatchmakeSessionEx)
	protocol.SetHandlerGetSimpleCommunity(commonProtocol.getSimpleCommunity)
	protocol.SetHandlerUpdateNotificationData(commonProtocol.updateNotificationData)
	protocol.SetHandlerGetFriendNotificationData(commonProtocol.getFriendNotificationData)
	protocol.SetHandlerGetlstFriendNotificationData(commonProtocol.getlstFriendNotificationData)
	protocol.SetHandlerAddToBlockList(commonProtocol.addToBlockList)
	protocol.SetHandlerRemoveFromBlockList(commonProtocol.removeFromBlockList)
	protocol.SetHandlerGetMyBlockList(commonProtocol.getMyBlockList)

	endpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		commonProtocol.manager.Mutex.Lock()
		database.InactivateNotificationDatas(commonProtocol.manager, connection.PID())
		commonProtocol.manager.Mutex.Unlock()
	})

	return commonProtocol
}
