package matchmake_extension

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"golang.org/x/exp/slices"
)

func remove(l []*nex.PID, p *nex.PID) []*nex.PID {
	for i, other := range l {
		if other.Value() == p.Value() {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

func includes(l []*nex.PID, p *nex.PID) bool {
	for _, other := range l {
		if other.Value() == p.Value() {
			return true
		}
	}

	return false
}

func getSimplePlayingSession(err error, packet nex.PacketInterface, callID uint32, listPID []*nex.PID, includeLoginUser bool) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender().(*nex.PRUDPClient)
	server := commonMatchmakeExtensionProtocol.server

	if includes(listPID, client.PID()) {
		listPID = remove(listPID, client.PID())
	}

	if includeLoginUser && !includes(listPID, client.PID()) {
		listPID = append(listPID, client.PID())
	}

	simplePlayingSessions := make(map[string]*match_making_types.SimplePlayingSession)

	for gatheringID, session := range common_globals.Sessions {
		for _, pid := range listPID {
			key := fmt.Sprintf("%d-%d", gatheringID, pid.Value())
			if simplePlayingSessions[key] == nil {
				connectedPIDs := make([]uint64, 0)
				for _, connectionID := range session.ConnectionIDs {
					player := server.FindClientByConnectionID(connectionID)
					if player == nil {
						common_globals.Logger.Warning("Player not found")
						continue
					}

					connectedPIDs = append(connectedPIDs, player.PID().Value())
				}

				if slices.Contains(connectedPIDs, pid.Value()) {
					simplePlayingSessions[key] = match_making_types.NewSimplePlayingSession()
					simplePlayingSessions[key].PrincipalID = pid
					simplePlayingSessions[key].GatheringID = gatheringID
					simplePlayingSessions[key].GameMode = session.GameMatchmakeSession.GameMode
					simplePlayingSessions[key].Attribute0 = session.GameMatchmakeSession.Attributes[0]
				}
			}
		}
	}

	lstSimplePlayingSession := make([]*match_making_types.SimplePlayingSession, 0)

	for _, simplePlayingSession := range simplePlayingSessions {
		lstSimplePlayingSession = append(lstSimplePlayingSession, simplePlayingSession)
	}

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteListStructure(lstSimplePlayingSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodGetSimplePlayingSession
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
