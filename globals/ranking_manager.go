package common_globals

import (
	"database/sql"

	"github.com/PretendoNetwork/nex-go/v2"
)

// RankingManager manages ranking storage
type RankingManager struct {
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint
}

// NewRankingManager returns a new StorageManagerManager
func NewRankingManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *RankingManager {
	rm := &RankingManager{
		Endpoint: endpoint,
		Database: db,
	}

	return rm
}
