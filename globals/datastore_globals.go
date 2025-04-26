package common_globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
)

// DataStoreManager manages a DataStore instance
type DataStoreManager struct {
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint
}

// NewDataStoreManager returns a new DataStoreManager
func NewDataStoreManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *DataStoreManager {
	return &DataStoreManager{
		Database: db,
		Endpoint: endpoint,
	}
}
