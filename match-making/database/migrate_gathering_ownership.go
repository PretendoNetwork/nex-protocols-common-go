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
func MigrateGatheringOwnership(manager *common_globals.MatchmakingManager, connection *nex.PRUDPConnection, gathering match_making_types.Gathering, participants []uint64, candidates []uint64, disconnect bool) (uint64, *nex.Error) {
	var nexError *nex.Error
	var uniqueParticipants []uint64
	var newOwner uint64

	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	if len(candidates) != 0 {
		// * Candidates must be connected
		for _, candidate := range candidates {
			if endpoint.FindConnectionByPID(candidate) != nil {
				uniqueParticipants = append(uniqueParticipants, candidate)
			}
		}
	} else {
		uniqueParticipants = common_globals.RemoveDuplicates(participants)
	}

	var previousOwnerFound bool = false
	for _, participant := range uniqueParticipants {
		if participant != uint64(gathering.OwnerPID) {
			newOwner = participant
			break
		} else {
			previousOwnerFound = true
		}
	}

	if !disconnect {
		// * There are no candidates available
		if len(participants) == 0 && len(candidates) == 0 {
			return 0, nex.NewError(nex.ResultCodes.RendezVous.NotParticipatedGathering, "change_error")
		}

		if newOwner == 0 {
			// * If there were candidates given which weren't the previous owner but no owner was selected, all of them must be offline
			if len(candidates) != 0 && !previousOwnerFound {
				return 0, nex.NewError(nex.ResultCodes.RendezVous.UserIsOffline, "change_error")
			} else {
				return 0, nil
			}
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
		oEvent.Type = types.NewUInt32(notifications.BuildNotificationType(category, subtype))
		oEvent.Param1 = gathering.ID

		common_globals.SendNotificationEvent(endpoint, oEvent, uniqueParticipants)
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
	oEvent.Param1 = gathering.ID
	oEvent.Param2 = types.NewUInt32(uint32(newOwner)) // TODO - This assumes a legacy client. Will not work on the Switch

	// TODO - StrParam doesn't have this value on some servers
	// * https://github.com/kinnay/NintendoClients/issues/101
	// * unixTime := time.Now()
	// * oEvent.StrParam = strconv.FormatInt(unixTime.UnixMicro(), 10)

	common_globals.SendNotificationEvent(endpoint, oEvent, uniqueParticipants)
	return newOwner, nil
}
