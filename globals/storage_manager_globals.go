package common_globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
)

// StorageManagerManager manages storage manager management. factory.
type StorageManagerManager struct {
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint
}

// NewStorageManagerManager returns a new StorageManagerManager
func NewStorageManagerManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *StorageManagerManager {
	smm := &StorageManagerManager{
		Endpoint: endpoint,
		Database: db,
	}

	return smm
}
