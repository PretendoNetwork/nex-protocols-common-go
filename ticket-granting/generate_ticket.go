package ticket_granting

import (
	"crypto/rand"

	"github.com/PretendoNetwork/nex-go"
)

func generateTicket(userPID uint32, targetPID uint32) ([]byte, uint32) {
	passwordFromPIDHandler := commonTicketGrantingProtocol.server.PasswordFromPIDFunction()
	if passwordFromPIDHandler == nil {
		logger.Warning("Missing passwordFromPIDHandler!")
		return []byte{}, nex.Errors.Core.Unknown
	}

	var userPassword string
	var targetPassword string
	var errorCode uint32

	// TODO - Maybe we should error out if the user PID is the server account?
	switch userPID {
	case 2: // "Quazal Rendez-Vous" (the server user) account
		userPassword = commonTicketGrantingProtocol.server.KerberosPassword()
	case 100: // guest user account
		userPassword = "MMQea3n!fsik"
	default:
		userPassword, errorCode = passwordFromPIDHandler(userPID)
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	switch targetPID {
	case 2: // "Quazal Rendez-Vous" (the server user) account
		targetPassword = commonTicketGrantingProtocol.server.KerberosPassword()
	case 100: // guest user account
		targetPassword = "MMQea3n!fsik"
	default:
		targetPassword, errorCode = passwordFromPIDHandler(userPID)
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	userKey := nex.DeriveKerberosKey(userPID, []byte(userPassword))
	targetKey := nex.DeriveKerberosKey(targetPID, []byte(targetPassword))
	sessionKey := make([]byte, commonTicketGrantingProtocol.server.KerberosKeySize())
	_, err := rand.Read(sessionKey)
	if err != nil {
		logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticketInternalData := nex.NewKerberosTicketInternalData()
	serverTime := nex.NewDateTime(0)
	serverTime.UTC()
	ticketInternalData.SetTimestamp(serverTime)
	ticketInternalData.SetUserPID(userPID)
	ticketInternalData.SetSessionKey(sessionKey)

	encryptedTicketInternalData, err := ticketInternalData.Encrypt(targetKey, nex.NewStreamOut(commonTicketGrantingProtocol.server))
	if err != nil {
		logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticket := nex.NewKerberosTicket()
	ticket.SetSessionKey(sessionKey)
	ticket.SetTargetPID(targetPID)
	ticket.SetInternalData(encryptedTicketInternalData)

	encryptedTicket, err := ticket.Encrypt(userKey, nex.NewStreamOut(commonTicketGrantingProtocol.server))
	if err != nil {
		logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	return encryptedTicket, 0
}
