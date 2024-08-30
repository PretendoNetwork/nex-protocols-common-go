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

func (commonProtocol *CommonProtocol) updateSessionHostV1(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	commonProtocol.manager.Mutex.Lock()

	gathering, _, participants, _, nexError := database.FindGatheringByID(commonProtocol.manager, gid.Value)
	if nexError != nil {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nexError
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if !slices.Contains(participants, connection.PID().Value()) {
		commonProtocol.manager.Mutex.Unlock()
		return nil, nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	if gathering.Flags.PAND(match_making.GatheringFlags.ParticipantsChangeOwner) == 0 {
		nexError = database.UpdateSessionHost(commonProtocol.manager, gid.Value, gathering.OwnerPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeHost(commonProtocol.manager.Database, connection.PID(), gid.Value, gathering.HostPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}
	} else {
		nexError = database.UpdateSessionHost(commonProtocol.manager, gid.Value, connection.PID(), connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeHost(commonProtocol.manager.Database, connection.PID(), gid.Value, gathering.HostPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		nexError = tracking.LogChangeOwner(commonProtocol.manager.Database, connection.PID(), gid.Value, gathering.OwnerPID, connection.PID())
		if nexError != nil {
			commonProtocol.manager.Mutex.Unlock()
			return nil, nexError
		}

		category := notifications.NotificationCategories.OwnershipChanged
		subtype := notifications.NotificationSubTypes.OwnershipChanged.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID()
		oEvent.Type.Value = notifications.BuildNotificationType(category, subtype)
		oEvent.Param1.Value = gid.Value
		oEvent.Param2.Value = connection.PID().LegacyValue() // TODO - This assumes a legacy client. Will not work on the Switch

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
