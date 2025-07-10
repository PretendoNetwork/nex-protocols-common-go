package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/tracking"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

// MigrateGatheringOwnership switches the owner of the gathering with a different one
func MigrateGatheringOwnership(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, gathering match_making_types.Gathering, participants []uint64) (uint64, *nex.Error) {
	var nexError *nex.Error
	var uniqueParticipants []uint64 = common_globals.RemoveDuplicates(participants)
	var newOwner uint64
	for _, participant := range uniqueParticipants {
		if participant != uint64(gathering.OwnerPID) {
			newOwner = participant
			break
		}
	}

	// * We couldn't find a new owner, so we unregister the gathering
	if newOwner == 0 {
		nexError = UnregisterGathering(manager, connection.PID(), uint32(gathering.ID))
		if nexError != nil {
			return 0, nexError
		}

		category := notifications.NotificationCategories.GatheringUnregistered
		subtype := notifications.NotificationSubTypes.GatheringUnregistered.None

		oEvent := notifications_types.NewNotificationEvent()
		oEvent.PIDSource = connection.PID().Copy().(types.PID)
		oEvent.Type = types.UInt32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1 = types.UInt64(gathering.ID)

		common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, uniqueParticipants)
		return 0, nil
	}

	oldOwner := gathering.OwnerPID.Copy().(types.PID)

	// * Set the new owner
	gathering.OwnerPID = types.NewPID(newOwner)

	nexError = UpdateSessionHost(manager, uint32(gathering.ID), gathering.OwnerPID, gathering.HostPID)
	if nexError != nil {
		return 0, nexError
	}

	nexError = tracking.LogChangeOwner(manager.Database, connection.PID(), uint32(gathering.ID), oldOwner, gathering.OwnerPID)
	if nexError != nil {
		return 0, nexError
	}

	category := notifications.NotificationCategories.OwnershipChanged
	subtype := notifications.NotificationSubTypes.OwnershipChanged.None

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(types.PID)
	oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
	oEvent.Param1 = types.UInt64(gathering.ID)
	oEvent.Param2 = types.UInt64(newOwner)

	// TODO - StrParam doesn't have this value on some servers
	// * https://github.com/kinnay/NintendoClients/issues/101
	// * unixTime := time.Now()
	// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	common_globals.SendNotificationEvent(connection.Endpoint().(*nex.PRUDPEndPoint), oEvent, uniqueParticipants)
	return newOwner, nil
}
