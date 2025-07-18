package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	_ "github.com/PretendoNetwork/nex-protocols-go/v2"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

type CommonProtocol struct {
	endpoint                   *nex.PRUDPEndPoint
	protocol                   match_making.Interface
	manager                    *common_globals.MatchmakingManager
	OnAfterUnregisterGathering func(packet nex.PacketInterface, idGathering types.UInt32)
	OnAfterFindBySingleID      func(packet nex.PacketInterface, id types.UInt32)
	OnAfterUpdateSessionURL    func(packet nex.PacketInterface, idGathering types.UInt32, strURL types.String)
	OnAfterUpdateSessionHostV1 func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterGetSessionURLs      func(packet nex.PacketInterface, gid types.UInt32)
	OnAfterUpdateSessionHost   func(packet nex.PacketInterface, gid types.UInt32, isMigrateOwner types.Bool)
}

// SetManager defines the matchmaking manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.MatchmakingManager) {
	var err error

	commonProtocol.manager = manager

	manager.GetDetailedGatheringByID = database.GetDetailedGatheringByID

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS matchmaking`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS matchmaking.gatherings (
		id bigserial PRIMARY KEY,
		owner_pid numeric(20),
		host_pid numeric(20),
		min_participants integer,
		max_participants integer,
		participation_policy bigint,
		policy_argument bigint,
		flags bigint,
		state bigint,
		description text,
		registered boolean NOT NULL DEFAULT true,
		type text NOT NULL DEFAULT '',
		started_time timestamp,
		participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[]
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS tracking`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.register_gathering (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.join_gathering (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint,
		new_participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[],
		total_participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[]
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.leave_gathering (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint,
		total_participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[]
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.disconnect_gathering (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint,
		total_participants numeric(20)[] NOT NULL DEFAULT array[]::numeric(20)[]
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.unregister_gathering (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.change_host (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint,
		old_host_pid numeric(20),
		new_host_pid numeric(20)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS tracking.change_owner (
		id bigserial PRIMARY KEY,
		date timestamp,
		source_pid numeric(20),
		gathering_id bigint,
		old_owner_pid numeric(20),
		new_owner_pid numeric(20)
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making.Interface) *CommonProtocol {
	endpoint := protocol.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol := &CommonProtocol{
		endpoint: endpoint,
		protocol: protocol,
	}

	protocol.SetHandlerUnregisterGathering(commonProtocol.unregisterGathering)
	protocol.SetHandlerFindBySingleID(commonProtocol.findBySingleID)
	protocol.SetHandlerUpdateSessionURL(commonProtocol.updateSessionURL)
	protocol.SetHandlerUpdateSessionHostV1(commonProtocol.updateSessionHostV1)
	protocol.SetHandlerGetSessionURLs(commonProtocol.getSessionURLs)
	protocol.SetHandlerUpdateSessionHost(commonProtocol.updateSessionHost)

	endpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		commonProtocol.manager.Mutex.Lock()
		database.DisconnectParticipant(commonProtocol.manager, connection)
		commonProtocol.manager.Mutex.Unlock()
	})

	return commonProtocol
}
