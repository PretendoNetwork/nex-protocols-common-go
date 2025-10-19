package matchmaking

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

func (commonProtocol *CommonProtocol) unregisterGathering(err error, packet nex.PacketInterface, callID uint32, idGathering types.UInt32) (*nex.RMCMessage, *nex.Error) {
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

	if !gathering.OwnerPID.Equals(connection.PID()) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	nexError = database.UnregisterGathering(commonProtocol.manager, connection.PID(), uint32(idGathering))
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	category := notifications.NotificationCategories.GatheringUnregistered
	subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(types.PID)
	oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = types.UInt64(idGathering)

	common_globals.SendNotificationEvent(endpoint, oEvent, common_globals.RemoveDuplicates(participants))

	commonProtocol.manager.Mutex.Unlock()

	retval := types.NewBool(true)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUnregisterGathering
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUnregisterGathering != nil {
		go commonProtocol.OnAfterUnregisterGathering(packet, idGathering)
	}

	return rmcResponse, nil
}
