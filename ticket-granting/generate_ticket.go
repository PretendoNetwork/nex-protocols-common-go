package ticket_granting

import (
	"crypto/rand"

	"github.com/PretendoNetwork/nex-go"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func generateTicket(userPID *nex.PID, targetPID *nex.PID) ([]byte, uint32) {
	// TODO - Remove cast to PRUDPServer once websockets are implemented
	server := commonTicketGrantingProtocol.server.(*nex.PRUDPServer)

	passwordFromPIDHandler := server.PasswordFromPID
	if passwordFromPIDHandler == nil {
		common_globals.Logger.Warning("Server is missing PasswordFromPID handler!")
		return []byte{}, nex.Errors.Core.Unknown
	}

	var userPassword []byte
	var targetPassword []byte
	var errorCode uint32

	// TODO - Maybe we should error out if the user PID is the server account?
	switch userPID.Value() {
	case 2: // * "Quazal Rendez-Vous" (the server user) account. Used as the Kerberos target
		userPassword = server.KerberosPassword()
	case 100: // * Guest user account. Used when creating a new NEX account
		userPassword = []byte("MMQea3n!fsik")
	default:
		password, err := passwordFromPIDHandler(userPID)
		userPassword = []byte(password)
		errorCode = err
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	switch targetPID.Value() {
	case 2: // * "Quazal Rendez-Vous" (the server user) account. Used as the Kerberos target
		targetPassword = server.KerberosPassword()
	case 100: // * Guest user account. Used when creating a new NEX account
		targetPassword = []byte("MMQea3n!fsik")
	default:
		password, err := passwordFromPIDHandler(userPID)
		targetPassword = []byte(password)
		errorCode = err
	}

	if errorCode != 0 {
		return []byte{}, errorCode
	}

	userKey := nex.DeriveKerberosKey(userPID, []byte(userPassword))
	targetKey := nex.DeriveKerberosKey(targetPID, []byte(targetPassword))
	sessionKey := make([]byte, server.KerberosKeySize())
	_, err := rand.Read(sessionKey)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticketInternalData := nex.NewKerberosTicketInternalData()
	serverTime := nex.NewDateTime(0).Now()

	ticketInternalData.Issued = serverTime
	ticketInternalData.SourcePID = userPID
	ticketInternalData.SessionKey = sessionKey

	encryptedTicketInternalData, err := ticketInternalData.Encrypt(targetKey, nex.NewStreamOut(commonTicketGrantingProtocol.server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticket := nex.NewKerberosTicket()
	ticket.SessionKey = sessionKey
	ticket.TargetPID = targetPID
	ticket.InternalData = encryptedTicketInternalData

	encryptedTicket, err := ticket.Encrypt(userKey, nex.NewStreamOut(commonTicketGrantingProtocol.server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	return encryptedTicket, 0
}
