package matchmake_extension

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/match-making/types"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
	"golang.org/x/exp/slices"
)

// * https://stackoverflow.com/a/70808522
func remove[T comparable](l []T, item T) []T {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}

func getSimplePlayingSession(err error, packet nex.PacketInterface, callID uint32, listPID []uint32, includeLoginUser bool) uint32 {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	client := packet.Sender()
	server := commonMatchmakeExtensionProtocol.server

	if slices.Contains(listPID, client.PID()) {
		listPID = remove(listPID, client.PID())
	}

	if includeLoginUser && !slices.Contains(listPID, client.PID()) {
		listPID = append(listPID, client.PID())
	}

	simplePlayingSessions := make(map[string]*match_making_types.SimplePlayingSession)

	for gatheringID, session := range common_globals.Sessions {
		for _, pid := range listPID {
			key := fmt.Sprintf("%d-%d", gatheringID, pid)
			if simplePlayingSessions[key] == nil {
				connectedPIDs := make([]uint32, 0)
				for _, connectionID := range session.ConnectionIDs {
					player := server.FindClientFromConnectionID(connectionID)
					if player == nil {
						common_globals.Logger.Warning("Player not found")
						continue
					}

					connectedPIDs = append(connectedPIDs, player.PID())
				}

				if slices.Contains(connectedPIDs, pid) {
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

	rmcResponse := nex.NewRMCResponse(matchmake_extension.ProtocolID, callID)
	rmcResponse.SetSuccess(matchmake_extension.MethodGetSimplePlayingSession, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(packet.Destination())
	responsePacket.SetDestination(packet.Source())
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)

	return 0
}
