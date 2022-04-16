package authentication

import (
	"crypto/rand"

	"github.com/PretendoNetwork/nex-go"
)

func generateTicket(userPID uint32, targetPID uint32) ([]byte, uint32) {
	if commonAuthenticationProtocol.passwordFromPIDHandler == nil {
		logger.Warning("Missing passwordFromPIDHandler!")
		return []byte{}, nex.Errors.Core.Unknown
	}

	var userPassword string
	var serverPassword string
	var errorCode uint32

	if userPID == 2 { // "Quazal Rendez-Vous", AKA server account
		userPassword = commonAuthenticationProtocol.server.KerberosPassword()
	} else if userPID == 100 { // "guest" account
		userPassword = "MMQea3n!fsik"
	} else {
		userPassword, errorCode = commonAuthenticationProtocol.passwordFromPIDHandler(userPID)
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	if targetPID == 2 { // "Quazal Rendez-Vous", AKA server account
		userPassword = commonAuthenticationProtocol.server.KerberosPassword()
	} else if targetPID == 100 { // "guest" account
		userPassword = "MMQea3n!fsik"
	} else {
		userPassword, errorCode = commonAuthenticationProtocol.passwordFromPIDHandler(targetPID)
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	userKey := nex.DeriveKerberosKey(userPID, []byte(userPassword))
	serverKey := nex.DeriveKerberosKey(targetPID, []byte(serverPassword))
	sessionKey := make([]byte, commonAuthenticationProtocol.server.KerberosKeySize())
	rand.Read(sessionKey)

	ticketInternalData := nex.NewKerberosTicketInternalData()
	ticketInternalData.SetTimestamp(nex.NewDateTime(0)) // CHANGE THIS
	ticketInternalData.SetUserPID(userPID)
	ticketInternalData.SetSessionKey(sessionKey)
	encryptedTicketInternalData := ticketInternalData.Encrypt(serverKey, nex.NewStreamOut(commonAuthenticationProtocol.server))

	ticket := nex.NewKerberosTicket()
	ticket.SetSessionKey(sessionKey)
	ticket.SetTargetPID(targetPID)
	ticket.SetInternalData(encryptedTicketInternalData)

	return ticket.Encrypt(userKey, nex.NewStreamOut(commonAuthenticationProtocol.server)), 0
}
