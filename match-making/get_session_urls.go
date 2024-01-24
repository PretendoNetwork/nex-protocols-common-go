package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func getSessionURLs(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.ResultCodesRendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	hostPID := session.GameMatchmakeSession.Gathering.HostPID
	host := endpoint.FindConnectionByPID(hostPID.Value())
	if host == nil {
		// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
		common_globals.Logger.Warning("Host client not found, trying with owner client")
		host = endpoint.FindConnectionByPID(session.GameMatchmakeSession.Gathering.OwnerPID.Value())
		if host == nil {
			// * This popped up once during testing. Leaving it noted here in case it becomes a problem.
			common_globals.Logger.Error("Owner client not found")
			return nil, nex.ResultCodesCore.Exception
		}
	}

	rmcResponseStream := nex.NewByteStreamOut(server)

	host.StationURLs.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
