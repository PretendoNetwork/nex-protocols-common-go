package common_globals

import (
	"math"
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/PretendoNetwork/nex-go/v2/types"
	match_making_types "github.com/PretendoNetwork/nex-protocols-go/v2/match-making/types"
	notifications "github.com/PretendoNetwork/nex-protocols-go/v2/notifications"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"
)

var OutgoingCallID *nex.Counter[uint32] = nex.NewCounter[uint32](0)

// DeleteIndex removes a value from a slice with the given index
func DeleteIndex(s []uint32, index int) []uint32 {
	s[index] = s[len(s)-1]
	return s[:len(s)-1]
}

// CheckValidParticipant validates if a participant isn't an additional participant
func CheckValidParticipant(participant uint64) bool {
	// * Additional participants are stored as the negative value of the parent participant.
	// * This seems to only be possible on the 3DS and Wii U, so we don't have to check the uint64 range
	return (participant <= math.MaxInt32) || (participant > math.MaxUint32)
}

// CheckValidGathering checks if a Gathering is valid
func CheckValidGathering(gathering *match_making_types.Gathering) bool {
	if len(gathering.Description.Value) > 256 {
		return false
	}

	return true
}

// CheckValidMatchmakeSession checks if a MatchmakeSession is valid
func CheckValidMatchmakeSession(matchmakeSession *match_making_types.MatchmakeSession) bool {
	if !CheckValidGathering(matchmakeSession.Gathering) {
		return false
	}

	if matchmakeSession.Attributes.Length() != 6 {
		return false
	}

	if matchmakeSession.ProgressScore.Value > 100 {
		return false
	}

	if len(matchmakeSession.UserPassword.Value) > 32 {
		return false
	}

	// * Except for UserPassword, all strings must have a length lower than 256
	if len(matchmakeSession.CodeWord.Value) > 256 {
		return false
	}

	// * All buffers must have a length lower than 512
	if len(matchmakeSession.ApplicationBuffer.Value) > 512 {
		return false
	}

	if len(matchmakeSession.SessionKey.Value) > 512 {
		return false
	}

	return true
}

// CanJoinMatchmakeSession checks if a PID is allowed to join a matchmake session
func CanJoinMatchmakeSession(pid *types.PID, matchmakeSession *match_making_types.MatchmakeSession) *nex.Error {
	// TODO - Is this the right error?
	if !matchmakeSession.OpenParticipation.Value {
		return nex.NewError(nex.ResultCodes.RendezVous.PermissionDenied, "change_error")
	}

	// * Only allow friends
	// TODO - This won't work on Switch!
	if matchmakeSession.ParticipationPolicy.Value == 98 {
		if GetUserFriendPIDsHandler == nil {
			Logger.Warning("Missing GetUserFriendPIDsHandler!")
			return nex.NewError(nex.ResultCodes.Core.NotImplemented, "change_error")
		}

		friendList := GetUserFriendPIDsHandler(pid.LegacyValue())
		if !slices.Contains(friendList, matchmakeSession.OwnerPID.LegacyValue()) {
			return nex.NewError(nex.ResultCodes.RendezVous.NotFriend, "change_error")
		}
	}

	return nil
}

// SendNotificationEvent sends a notification event to the specified targets
func SendNotificationEvent(endpoint *nex.PRUDPEndPoint, event *notifications_types.NotificationEvent, targets []uint64) {
	server := endpoint.Server
	stream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	event.WriteTo(stream)

	rmcRequest := nex.NewRMCRequest(endpoint)
	rmcRequest.ProtocolID = notifications.ProtocolID
	rmcRequest.CallID = OutgoingCallID.Next()
	rmcRequest.MethodID = notifications.MethodProcessNotificationEvent
	rmcRequest.Parameters = stream.Bytes()

	rmcRequestBytes := rmcRequest.Bytes()

	for _, pid := range targets {
		if !CheckValidParticipant(pid) {
			continue
		}

		target := endpoint.FindConnectionByPID(pid)
		if target == nil {
			Logger.Warning("Client not found")
			continue
		}

		var messagePacket nex.PRUDPPacketInterface

		if target.DefaultPRUDPVersion == 0 {
			messagePacket, _ = nex.NewPRUDPPacketV0(server, target, nil)
		} else {
			messagePacket, _ = nex.NewPRUDPPacketV1(server, target, nil)
		}

		messagePacket.SetType(constants.DataPacket)
		messagePacket.AddFlag(constants.PacketFlagNeedsAck)
		messagePacket.AddFlag(constants.PacketFlagReliable)
		messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
		messagePacket.SetSourceVirtualPortStreamID(endpoint.StreamID)
		messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
		messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
		messagePacket.SetPayload(rmcRequestBytes)

		server.Send(messagePacket)
	}
}
