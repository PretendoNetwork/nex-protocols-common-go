package common_globals

import (
	"database/sql"
	"unicode/utf8"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	messaging_types "github.com/PretendoNetwork/nex-protocols-go/v2/messaging/types"
)

// MessagingManager manages messaging communications
type MessagingManager struct {
	Database           *sql.DB
	Endpoint           *nex.PRUDPEndPoint

	MatchmakingManager *MatchmakingManager
	ValidateMessage    func(message types.DataHolder) (types.UInt64, types.UInt32, *nex.Error)
	GetMessageHeader   func(message types.DataHolder) (messaging_types.UserMessage, *nex.Error)
	SetMessageHeader   func(message types.DataHolder, header messaging_types.UserMessage) (types.DataHolder, *nex.Error)
	ProcessMessage     func(message types.DataHolder, recipientID types.UInt64, recipientType types.UInt32, sendMessage bool) (types.DataHolder, types.List[types.UInt32], types.List[types.PID], *nex.Error)
}

// ValidateUserMessage checks if a UserMessage is valid, and returns its validity and the recipient information of the message
func (mm *MessagingManager) ValidateUserMessage(userMessage messaging_types.UserMessage) (types.UInt64, types.UInt32, *nex.Error) {
	if utf8.RuneCountInString(string(userMessage.StrSubject)) > 256 {
		return 0, 0, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Message subject is too long")
	}

	//  Check that the specified recipient is valid
	recipientID, recipientType := GetUserMessageRecipientData(mm.Endpoint.Server.LibraryVersions.Messaging, userMessage)

	return recipientID, recipientType, nil
}

// ValidateTextMessage checks if a TextMessage is valid, and returns its validity and the recipient information of the message
func (mm *MessagingManager) ValidateTextMessage(textMessage messaging_types.TextMessage) (types.UInt64, types.UInt32, *nex.Error) {
	recipientID, recipientType, nexError := mm.ValidateUserMessage(textMessage.UserMessage)
	if nexError != nil {
		return 0, 0, nexError
	}

	if utf8.RuneCountInString(string(textMessage.StrTextBody)) > 256 {
		return 0, 0, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Message text body is too long")
	}

	return recipientID, recipientType, nil
}

// ValidateBinaryMessage checks if a BinaryMessage is valid, and returns its validity and the recipient information of the message
func (mm *MessagingManager) ValidateBinaryMessage(binaryMessage messaging_types.BinaryMessage) (types.UInt64, types.UInt32, *nex.Error) {
	recipientID, recipientType, nexError := mm.ValidateUserMessage(binaryMessage.UserMessage)
	if nexError != nil {
		return 0, 0, nexError
	}

	if len(binaryMessage.BinaryBody) > 512 {
		return 0, 0, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Message binary body is too long")
	}

	return recipientID, recipientType, nil
}

// DefaultValidateMessage is the default handler of MessagingManager to validate an input message holder
func (mm *MessagingManager) DefaultValidateMessage(message types.DataHolder) (types.UInt64, types.UInt32, *nex.Error) {
	messageObjectID := message.Object.ObjectID()
	if messageObjectID.Equals(types.NewString("TextMessage")) {
		return mm.ValidateTextMessage(message.Object.(messaging_types.TextMessage))
	} else if messageObjectID.Equals(types.NewString("BinaryMessage")) {
		return mm.ValidateBinaryMessage(message.Object.(messaging_types.BinaryMessage))
	}

	return 0, 0, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid data holder object ID")
}

// DefaultGetMessageHeader is the default handler for GetMessageHeader
func (mm *MessagingManager) DefaultGetMessageHeader(message types.DataHolder) (messaging_types.UserMessage, *nex.Error) {
	messageObjectID := message.Object.ObjectID()
	if messageObjectID.Equals(types.NewString("TextMessage")) {
		return message.Object.(messaging_types.TextMessage).UserMessage, nil
	} else if messageObjectID.Equals(types.NewString("BinaryMessage")) {
		return message.Object.(messaging_types.BinaryMessage).UserMessage, nil
	}

	return messaging_types.UserMessage{}, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid data holder object ID")
}

// DefaultSetMessageHeader is the default handler for SetMessageHeader
func (mm *MessagingManager) DefaultSetMessageHeader(message types.DataHolder, header messaging_types.UserMessage) (types.DataHolder, *nex.Error) {
	messageObjectID := message.Object.ObjectID()
	if messageObjectID.Equals(types.NewString("TextMessage")) {
		textMessage := message.Object.(messaging_types.TextMessage)
		textMessage.UserMessage = header
		message.Object = textMessage
	} else if messageObjectID.Equals(types.NewString("BinaryMessage")) {
		binaryMessage := message.Object.(messaging_types.BinaryMessage)
		binaryMessage.UserMessage = header
		message.Object = binaryMessage
	} else {
		return message, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Invalid data holder object ID")
	}

	return message, nil
}

// PrepareMessage populates the required missing fields of the message
func (mm *MessagingManager) PrepareMessage(senderPID types.PID, message types.DataHolder) (types.DataHolder, *nex.Error) {
	messageHeader, nexError := mm.GetMessageHeader(message)
	if nexError != nil {
		return message, nexError
	}

	senderAccount, nexError := mm.Endpoint.AccountDetailsByPID(senderPID)
	if nexError != nil {
		return message, nexError
	}

	// * The message subject is capped to 60 characters
	if utf8.RuneCountInString(string(messageHeader.StrSubject)) > 60 {
		messageHeader.StrSubject = types.String(ResizeString(string(messageHeader.StrSubject), 60))
	}

	messageHeader.PIDSender = senderAccount.PID
	messageHeader.StrSender = types.String(senderAccount.Username)
	messageHeader.Receptiontime = messageHeader.Receptiontime.Now()

	message, nexError = mm.SetMessageHeader(message, messageHeader)
	if nexError != nil {
		return message, nexError
	}

	return message, nil
}

// NewMessagingManager returns a new MessagingManager
func NewMessagingManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *MessagingManager {
	mm := &MessagingManager{
		Endpoint: endpoint,
		Database: db,
	}

	mm.ValidateMessage = mm.DefaultValidateMessage
	mm.GetMessageHeader = mm.DefaultGetMessageHeader
	mm.SetMessageHeader = mm.DefaultSetMessageHeader

	return mm
}
