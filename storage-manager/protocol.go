package storage_manager

import (
	"github.com/PretendoNetwork/nex-go/v2"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	storagemanager "github.com/PretendoNetwork/nex-protocols-go/v2/storage-manager"
)

type CommonProtocol struct {
	endpoint nex.EndpointInterface
	protocol storagemanager.Interface
	manager  *commonglobals.StorageManagerManager
}

// SetManager defines the utility manager to be used by the common protocol
func (commonProtocol *CommonProtocol) SetManager(manager *commonglobals.StorageManagerManager) {
	var err error

	commonProtocol.manager = manager

	_, err = manager.Database.Exec(`CREATE SCHEMA IF NOT EXISTS storage_manager`)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return
	}

	_, err = manager.Database.Exec(`CREATE TABLE IF NOT EXISTS storage_manager.unique_ids (
    	/* Should be unsigned but since the server generates we can limit the range */
		unique_id serial4 PRIMARY KEY,
		slot_id int2 CONSTRAINT slot_range CHECK (slot_id >= 0 AND slot_id < 5),
		/* Basically a random value for users to get more slots.
		   Again we generate these and can limit the range. NULL for no card */
		card_id int8,
		associated_pid numeric(20),
		associated_time timestamp,
		creation_time timestamp,
		/* All IDs are unique on user, card, slot */
		UNIQUE NULLS NOT DISTINCT (associated_pid, card_id, slot_id)
	)`)
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return
	}
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol storagemanager.Interface) *CommonProtocol {
	commonProtocol := &CommonProtocol{
		endpoint: protocol.Endpoint(),
		protocol: protocol,
	}

	protocol.SetHandlerAcquireCardID(commonProtocol.acquireCardId)
	protocol.SetHandlerAcquireNexUniqueID(commonProtocol.acquireNexUniqueId)
	protocol.SetHandlerActivateWithCardID(commonProtocol.setHandlerActivateWithCardID)
	protocol.SetHandlerNexUniqueIDToPrincipalID(commonProtocol.nexUniqueIdToPrincipalId)

	return commonProtocol
}
