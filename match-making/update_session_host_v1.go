package matchmaking

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making "github.com/PretendoNetwork/nex-protocols-go/v2/match-making"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

func (commonProtocol *CommonProtocol) updateSessionHostV1(err error, packet nex.PacketInterface, callID uint32, gid types.UInt32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	gathering, _, participants, _, nexError := database.FindGatheringByID(commonProtocol.manager, uint32(gid))
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

	if uint32(gathering.Flags)&match_making.GatheringFlags.ParticipantsChangeOwner == 0 {
		nexError = database.UpdateSessionHost(commonProtocol.manager, uint32(gid), gathering.OwnerPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeHost(commonProtocol.manager.Database, connection.PID(), uint32(gid), gathering.HostPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	} else {
		nexError = database.UpdateSessionHost(commonProtocol.manager, uint32(gid), connection.PID(), connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeHost(commonProtocol.manager.Database, connection.PID(), uint32(gid), gathering.HostPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeOwner(commonProtocol.manager.Database, connection.PID(), uint32(gid), gathering.OwnerPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		category := notifications.NotificationCategories.OwnershipChanged
		subtype := notifications.NotificationSubTypes.OwnershipChanged.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type = types.UInt32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1 = types.UInt64(gid)
		oEvent.Param2 = types.UInt64(connection.PID())

		// TODO - StrParam doesn't have this value on some servers
		// * https://github.com/kinnay/NintendoClients/issues/101
		// * unixTime := time.Now()
		// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

		common_globals.SendNotificationEvent(endpoint, oEvent, common_globals.RemoveDuplicates(participants))
	}

	commonProtocol.manager.Mutex.Unlock()

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodUpdateSessionHostV1
	rmcResponse.CallID = callID

	if commonProtocol.OnAfterUpdateSessionHostV1 != nil {
		go commonProtocol.OnAfterUpdateSessionHostV1(packet, gid)
	}

	return rmcResponse, nil
}
