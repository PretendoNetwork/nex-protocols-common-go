package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	match_making_database "github.com/PretendoNetwork/nex-protocols-common-go/v2/match-making/database"
	messaging_constants "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/constants"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
)

// ProcessMessage delivers the given message and stores it in the server
func ProcessMessage(manager *common_globals.MessagingManager, message types.DataHolder, recipientIDs types.List[types.UInt64], recipientType messaging_constants.RecipientType, sendMessage bool) (types.DataHolder, types.List[types.UInt32], types.List[types.PID], *nex.Error) {
	header, nexError := manager.GetMessageHeader(message)
	if nexError != nil {
		return message, nil, nil, nexError
	}

	var targetConnections []*nex.PRUDPConnection
	var lstSandboxNodeID types.List[types.UInt32]
	var lstParticipants types.List[types.PID]

	switch recipientType {
	case 1: // * PID
		// TODO - Should we check that the PID exists with manager.Endpoint.AccountDetailsByPID?

		// * We don't have to get the connection if this isn't an instant message
		if header.UIFlags&1 != 0 {
			for _, recipientID := range recipientIDs {
				targetConnection := manager.Endpoint.FindConnectionByPID(uint64(recipientID))
				if targetConnection != nil {
					targetConnections = append(targetConnections, targetConnection)
					lstSandboxNodeID = append(lstSandboxNodeID, types.UInt32(targetConnection.ID))
					lstParticipants = append(lstParticipants, targetConnection.PID()) // * In official servers this has garbage values, but we will populate it properly
				}
			}
		}
	case 2: // * Gathering ID
		if manager.MatchmakingManager == nil {
			common_globals.Logger.Warning("MessagingManager.MatchmakingManager is not set!")
			return message, nil, nil, nex.NewError(nex.ResultCodes.Core.NotImplemented, "Messages to gatherings are not implemented")
		}

		// * DeliverMessageMultiTarget seems to only support PIDs based on how the method gets handled internally on games
		if len(recipientIDs) != 1 {
			return message, nil, nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Messages to multiple gatherings are not supported")
		}

		var recipientID uint32 = uint32(recipientIDs[0])

		// * Check that the gathering exists
		manager.MatchmakingManager.Mutex.RLock()
		_, _, participants, _, nexError := match_making_database.FindGatheringByID(manager.MatchmakingManager, recipientID)
		if nexError != nil {
			manager.MatchmakingManager.Mutex.RUnlock()
			return message, nil, nil, nexError
		}

		// * We don't have to get the participants connections if this isn't an instant message or we won't send it
		if header.UIFlags&1 != 0 && sendMessage {
			for _, participant := range participants {
				targetConnection := manager.Endpoint.FindConnectionByPID(participant)
				if targetConnection == nil {
					// * This shouldn't happen, but leaving it here just in case
					common_globals.Logger.Error("Participant in gathering not found in server")
					continue
				}

				targetConnections = append(targetConnections, targetConnection)
				lstSandboxNodeID = append(lstSandboxNodeID, types.UInt32(targetConnection.ID))
				lstParticipants = append(lstParticipants, targetConnection.PID())
			}
		}
		manager.MatchmakingManager.Mutex.RUnlock()
	default:
		return message, nil, nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid recipient type")
	}

	if header.UIFlags&1 != 0 { // * Instant message
		messageObjectID := message.Object.ObjectID()
		if messageObjectID.Equals(types.NewString("TextMessage")) {
			textMessage := message.Object.(messaging_types.TextMessage)
			for _, recipientID := range recipientIDs {
				nexError = InsertInstantTextMessage(manager, textMessage, recipientID, recipientType)
				if nexError != nil {
					return message, nil, nil, nexError
				}
			}
		} else if messageObjectID.Equals(types.NewString("BinaryMessage")) {
			binaryMessage := message.Object.(messaging_types.BinaryMessage)
			for _, recipientID := range recipientIDs {
				nexError = InsertInstantBinaryMessage(manager, binaryMessage, recipientID, recipientType)
				if nexError != nil {
					return message, nil, nil, nexError
				}
			}
		} else {
			return message, nil, nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid data holder object ID")
		}

		// * MessageDelivery will send the message if the user is connected, while Messaging will not
		if sendMessage {
			libraryVersion := manager.Endpoint.Server.LibraryVersions.Messaging

			// * If sending to PIDs, prepare the message for each of them if multiple targets are set
			if recipientType == 1 {
				for _, targetConnection := range targetConnections {
					targetHeader := common_globals.SetUserMessageRecipientData(libraryVersion, header, types.UInt64(targetConnection.PID()), recipientType)
					targetMessage, nexError := manager.SetMessageHeader(message, targetHeader)
					if nexError != nil {
						return message, nil, nil, nexError
					}

					common_globals.SendMessage(manager.Endpoint, targetMessage, []*nex.PRUDPConnection{targetConnection})
				}
			} else {
				common_globals.SendMessage(manager.Endpoint, message, targetConnections)
			}

		}
	} else {
		messageObjectID := message.Object.ObjectID()
		if messageObjectID.Equals(types.NewString("TextMessage")) {
			textMessage := message.Object.(messaging_types.TextMessage)
			for _, recipientID := range recipientIDs {
				nexError = InsertTextMessage(manager, textMessage, recipientID, recipientType)
				if nexError != nil {
					return message, nil, nil, nexError
				}
			}
		} else if messageObjectID.Equals(types.NewString("BinaryMessage")) {
			binaryMessage := message.Object.(messaging_types.BinaryMessage)
			for _, recipientID := range recipientIDs {
				nexError = InsertBinaryMessage(manager, binaryMessage, recipientID, recipientType)
				if nexError != nil {
					return message, nil, nil, nexError
				}
			}
		} else {
			return message, nil, nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid data holder object ID")
		}
	}

	return message, lstSandboxNodeID, lstParticipants, nil
}
