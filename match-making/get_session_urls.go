package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func getSessionURLs(err error, packet nex.PacketInterface, callID uint32, gid uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[gid]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	server := commonProtocol.server

	// TODO - Remove cast to PRUDPClient?
	client := packet.Sender().(*nex.PRUDPClient)

	hostPID := session.GameMatchmakeSession.Gathering.HostPID
	host := server.FindClientByPID(client.DestinationPort, client.DestinationStreamType, hostPID.Value())
	if host == nil {
		// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
		common_globals.Logger.Warning("Host client not found, trying with owner client")
		host = server.FindClientByPID(client.DestinationPort, client.DestinationStreamType, session.GameMatchmakeSession.Gathering.OwnerPID.Value())
		if host == nil {
			// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
			common_globals.Logger.Error("Owner client not found")
			return nil, nex.Errors.Core.Exception
		}
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteListStationURL(host.StationURLs)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
