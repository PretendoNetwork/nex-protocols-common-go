package ticket_granting

import (
	"crypto/rand"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func generateTicket(userPID *types.PID, targetPID *types.PID) ([]byte, uint32) {
	// TODO - I would like to remove this use of commonProtocol, if possible
	server := commonProtocol.server.(*nex.PRUDPServer)

	var userPassword []byte
	var targetPassword []byte
	var errorCode uint32

	// TODO - Maybe we should error out if the user PID is the server account?
	switch userPID.Value() {
	case 2: // * "Quazal Rendez-Vous" (the server user) account. Used as the Kerberos target
		userPassword = server.KerberosPassword()
	case 100: // * Guest user account. Used when creating a new NEX account
		userPassword = []byte("MMQea3n!fsik") // TODO - Configure this
	default:
		password, err := server.PasswordFromPID(userPID)
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
		password, err := server.PasswordFromPID(userPID)
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
	serverTime := types.NewDateTime(0).Now()

	ticketInternalData.Issued = serverTime
	ticketInternalData.SourcePID = userPID
	ticketInternalData.SessionKey = sessionKey

	encryptedTicketInternalData, err := ticketInternalData.Encrypt(targetKey, nex.NewByteStreamOut(server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticket := nex.NewKerberosTicket()
	ticket.SessionKey = sessionKey
	ticket.TargetPID = targetPID
	ticket.InternalData = types.NewBuffer(encryptedTicketInternalData)

	encryptedTicket, err := ticket.Encrypt(userKey, nex.NewByteStreamOut(server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	return encryptedTicket, 0
}
