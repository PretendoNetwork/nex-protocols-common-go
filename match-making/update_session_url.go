package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

func (commonProtocol *CommonProtocol) updateSessionURL(err error, packet nex.PacketInterface, callID uint32, idGathering *types.PrimitiveU32, strURL *types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	session, ok := common_globals.GetSession(idGathering.Value)
	if !ok {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// * Mario Kart 7 seems to set an empty strURL, so I assume this is what the method does?
	originalHost := session.GameMatchmakeSession.Gathering.HostPID
	session.GameMatchmakeSession.Gathering.HostPID = connection.PID().Copy().(*types.PID)

	if (common_globals.SessionManagementDebugLog) {
		common_globals.Logger.Infof("GID %d: UpdateSessionURL HOST from PID %d to PID %d", idGathering.Value, originalHost.LegacyValue(), connection.PID().LegacyValue())
	}

	retval := types.NewPrimitiveBool(true)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateSessionURL != nil {
		go commonProtocol.OnAfterUpdateSessionURL(packet, idGathering, strURL)
	}

	return rmcResponse, nil
}
