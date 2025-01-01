package matchmaking

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

func (commonProtocol *CommonProtocol) updateSessionURL(err error, packet nex.PacketInterface, callID uint32, idGathering types.UInt32, strURL types.String) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()
	gathering, _, participants, _, nexError := database.FindGatheringByID(commonProtocol.manager, uint32(idGathering))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if !slices.Contains(participants, uint64(connection.PID())) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	// TODO - Mario Kart 7 seems to set an empty strURL. What does that do if it's actually set?

	// * Only update the host
	nexError = database.UpdateSessionHost(commonProtocol.manager, uint32(idGathering), gathering.OwnerPID, connection.PID())
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	nexError = tracking.LogChangeHost(commonProtocol.manager.Database, connection.PID(), uint32(idGathering), gathering.HostPID, connection.PID())
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	retval := types.NewBool(true)

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
