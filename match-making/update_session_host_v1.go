package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func updateSessionHostV1(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if common_globals.FindConnectionSession(connection.ID) != gid.Value {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	session.GameMatchmakeSession.Gathering.HostPID = connection.PID()
	if session.GameMatchmakeSession.Gathering.Flags.PAND(match_making.GatheringFlags.DisconnectChangeOwner) != 0 {
		session.GameMatchmakeSession.Gathering.OwnerPID = connection.PID()
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUpdateSessionHostV1
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
