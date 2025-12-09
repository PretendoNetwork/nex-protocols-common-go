package common_globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

// UtilityManager manages NEX unique ids
type UtilityManager struct {
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint

	AllowUniqueIDStealing bool // Allows users' unique ids to be stolen while they are offline, likely unneeded but implemented for posterity

	GenerateNEXUniqueID             func(manager *UtilityManager, packet nex.PacketInterface) (types.UInt64, *nex.Error)
	GenerateNEXUniqueIDWithPassword func(manager *UtilityManager, packet nex.PacketInterface) (types.UInt64, types.UInt64, *nex.Error)
	GetIntegerSettings              func(manager *UtilityManager, packet nex.PacketInterface, integerSettingIndex uint32) (map[uint16]int32, *nex.Error)
	GetStringSettings               func(manager *UtilityManager, packet nex.PacketInterface, stringSettingIndex uint32) (map[uint16]string, *nex.Error)
}

// NewUtilityManager returns a new UtilityManager
func NewUtilityManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *UtilityManager {
	um := &UtilityManager{
		Endpoint:              endpoint,
		Database:              db,
		AllowUniqueIDStealing: false,
	}

	return um
}
