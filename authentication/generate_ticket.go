package authentication

import (
	"crypto/rand"
	"fmt"

	"github.com/PretendoNetwork/nex-go"
)

func generateTicket(userPID uint32, serverPID uint32) ([]byte, uint32) {
	if commonAuthenticationProtocol.passwordFromPIDHandler == nil {
		fmt.Println("[COMMON PROTOCOLS] Authentication::GenerateTicker missing passwordFromPIDHandler!")
		return []byte{}, nex.Errors.Core.Unknown
	}

	userPassword, errorCode := commonAuthenticationProtocol.passwordFromPIDHandler(userPID)
	if errorCode != 0 {
		return []byte{}, errorCode
	}

	serverPassword, errorCode := commonAuthenticationProtocol.passwordFromPIDHandler(serverPID)
	if errorCode != 0 {
		return []byte{}, errorCode
	}

	userKey := deriveKey(userPID, []byte(userPassword))
	serverKey := deriveKey(serverPID, []byte(serverPassword))
	sessionKey := make([]byte, commonAuthenticationProtocol.server.KerberosKeySize())
	rand.Read(sessionKey)

	ticketInternalData := nex.NewKerberosTicketInternalData()
	ticketInternalData.SetTimestamp(nex.NewDateTime(0)) // CHANGE THIS
	ticketInternalData.SetUserPID(userPID)
	ticketInternalData.SetSessionKey(sessionKey)
	encryptedTicketInternalData := ticketInternalData.Encrypt(serverKey, nex.NewStreamOut(commonAuthenticationProtocol.server))

	ticket := nex.NewKerberosTicket()
	ticket.SetSessionKey(sessionKey)
	ticket.SetTargetPID(serverPID)
	ticket.SetInternalData(encryptedTicketInternalData)

	return ticket.Encrypt(userKey, nex.NewStreamOut(commonAuthenticationProtocol.server)), 0
}

func deriveKey(pid uint32, password []byte) []byte {
	for i := 0; i < 65000+int(pid)%1024; i++ {
		password = nex.MD5Hash(password)
	}

	return password
}
