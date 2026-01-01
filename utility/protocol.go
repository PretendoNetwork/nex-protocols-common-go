package utility

import (
	"github.com/PretendoNetwork/nex-go/v2"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	utility_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/utility/database"
	utility "github.com/PretendoNetwork/nex-protocols-go/v2/utility"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol utility.Interface
	manager  *common_globals.UtilityManager
}

// SetManager defines the utility manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *common_globals.UtilityManager) {
	var err error

	commonProtocol.manager = manager

	if manager.GenerateNEXUniqueIDWithPassword == nil {
		manager.GenerateNEXUniqueIDWithPassword = utility_database.GenerateNEXUniqueIDWithPassword
	}

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS utility`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS utility.unique_ids (
		unique_id numeric(20) PRIMARY KEY,
		password numeric(20) NOT NULL DEFAULT 0,
		associated_pid numeric(20),
		associated_time timestamp,
		creation_time timestamp,
		is_primary_id bool NOT NULL DEFAULT false
	)`)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol utility.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerAcquireNexUniqueID(commonProtocol.acquireNexUniqueID)
	protocol.SetHandlerAcquireNexUniqueIDWithPassword(commonProtocol.acquireNexUniqueIDWithPassword)
	protocol.SetHandlerAssociateNexUniqueIDWithMyPrincipalID(commonProtocol.associateNexUniqueIDWithMyPrincipalID)
	protocol.SetHandlerAssociateNexUniqueIDsWithMyPrincipalID(commonProtocol.associateNexUniqueIDsWithMyPrincipalID)
	protocol.SetHandlerGetAssociatedNexUniqueIDWithMyPrincipalID(commonProtocol.getAssociatedNexUniqueIDWithMyPrincipalID)
	protocol.SetHandlerGetAssociatedNexUniqueIDsWithMyPrincipalID(commonProtocol.getAssociatedNexUniqueIDsWithMyPrincipalID)
	protocol.SetHandlerGetIntegerSettings(commonProtocol.getIntegerSettings)
	protocol.SetHandlerGetStringSettings(commonProtocol.getStringSettings)

	return commonProtocol
}
