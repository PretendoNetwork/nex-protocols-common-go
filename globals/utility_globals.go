package common_globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	utility_types "github.com/PretendoNetwork/nex-protocols-go/v2/utility/types"
)

// UtilityManager manages NEX unique IDs
type UtilityManager struct {
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint

	AllowUniqueIDStealing bool // Allows users' unique IDs to be stolen while they are offline, likely unneeded but implemented for posterity

	GenerateNEXUniqueIDWithPassword func(manager *UtilityManager, userPID types.PID, usePassword bool) (utility_types.UniqueIDInfo, *nex.Error)
	GetIntegerSettings              func(manager *UtilityManager, userPID types.PID, integerSettingIndex types.UInt32) (types.Map[types.UInt16, types.UInt32], *nex.Error)
	GetStringSettings               func(manager *UtilityManager, userPID types.PID, stringSettingIndex types.UInt32) (types.Map[types.UInt16, types.String], *nex.Error)
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
