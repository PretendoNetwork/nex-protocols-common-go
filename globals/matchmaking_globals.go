package common_globals

import (
	"sync"
)

var MatchmakingMutex *sync.RWMutex = &sync.RWMutex{}
var GetUserFriendPIDsHandler func(pid uint32) []uint32
