package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
)

func (commonProtocol *CommonProtocol) migrateGatheringOwnership(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32, lstPotentialNewOwnersID types.List[types.PID], participantsOnly types.Bool) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	// TODO - Is this actually unused?
	_ = participantsOnly

	commonProtocol.manager.Mutex.Lock()
	gathering, _, participants, _, nexError := database.FindGatheringByID(commonProtocol.manager, uint32(gid))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	// * Only the owner can use this method
	if connection.PID() != gathering.OwnerPID {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	candidates := make([]uint64, len(lstPotentialNewOwnersID))
	for i, candidate := range lstPotentialNewOwnersID {
		candidates[i] = uint64(candidate)
	}

	_, nexError = database.MigrateGatheringOwnership(commonProtocol.manager, connection, gathering, participants, candidates, false)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodMigrateGatheringOwnership
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterMigrateGatheringOwnership != nil {
		go commonProtocol.OnAfterMigrateGatheringOwnership(packet, gid, lstPotentialNewOwnersID, participantsOnly)
	}

	return rmcResponse, nil
}
