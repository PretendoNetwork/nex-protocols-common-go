package matchmaking

import (
	"fmt"

	"github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	_ "github.com/PretendoNetwork/nex-protocols-go"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

type CommonProtocol struct {
	endpoint *nex.PRUDPEndPoint
	protocol match_making.Interface
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol match_making.Interface) *CommonProtocol {
	endpoint := protocol.Endpoint().(*nex.PRUDPEndPoint)

	commonProtocol := &CommonProtocol{
		endpoint: endpoint,
		protocol: protocol,
	}

	common_globals.Sessions = make(map[uint32]*common_globals.CommonMatchmakeSession)

	protocol.SetHandlerUnregisterGathering(commonProtocol.unregisterGathering)
	protocol.SetHandlerFindBySingleID(commonProtocol.findBySingleID)
	protocol.SetHandlerUpdateSessionURL(commonProtocol.updateSessionURL)
	protocol.SetHandlerUpdateSessionHostV1(commonProtocol.updateSessionHostV1)
	protocol.SetHandlerGetSessionURLs(commonProtocol.getSessionURLs)
	protocol.SetHandlerUpdateSessionHost(commonProtocol.updateSessionHost)

	endpoint.OnConnectionEnded(func(connection *nex.PRUDPConnection) {
		fmt.Println("Leaving")
		common_globals.RemoveConnectionFromAllSessions(connection)
	})

	return commonProtocol
}
