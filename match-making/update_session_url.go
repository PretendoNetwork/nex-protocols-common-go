package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func updateSessionURL(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32, strURL *types.String) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodes.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering.Value]
	if !ok {
		return nil, nex.ResultCodes.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	// * Mario Kart 7 seems to set an empty strURL, so I assume this is what the method does?
	session.GameMatchmakeSession.Gathering.HostPID = connection.PID().Copy().(*types.PID)
	if session.GameMatchmakeSession.Gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) != 0 {
		session.GameMatchmakeSession.Gathering.OwnerPID = connection.PID().Copy().(*types.PID)
	}

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(server)

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
