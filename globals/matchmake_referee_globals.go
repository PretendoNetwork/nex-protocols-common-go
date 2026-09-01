package common_globals

import (
	"database/sql"
	"sync"

	"github.com/PretendoNetwork/nex-go/v2"
)

type MatchmakeRefereeManager struct {
    Database *sql.DB
    Endpoint *nex.PRUDPEndPoint
    Mutex    *sync.RWMutex
}

func NewMatchmakeRefereeManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *MatchmakeRefereeManager {
	return &MatchmakeRefereeManager{
		Endpoint: endpoint,
		Database: db,
		Mutex:    &sync.RWMutex{},
	}
}