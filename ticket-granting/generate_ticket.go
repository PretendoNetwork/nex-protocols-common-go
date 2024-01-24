package ticket_granting

import (
	"crypto/rand"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func generateTicket(source, target *nex.Account, sessionKeyLength int, server nex.ServerInterface) ([]byte, uint32) {
	if source == nil || target == nil {
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	sourceKey := nex.DeriveKerberosKey(source.PID, []byte(source.Password))
	targetKey := nex.DeriveKerberosKey(target.PID, []byte(target.Password))
	sessionKey := make([]byte, sessionKeyLength)

	_, err := rand.Read(sessionKey)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticketInternalData := nex.NewKerberosTicketInternalData()
	serverTime := types.NewDateTime(0).Now()

	ticketInternalData.Issued = serverTime
	ticketInternalData.SourcePID = source.PID
	ticketInternalData.SessionKey = sessionKey

	encryptedTicketInternalData, err := ticketInternalData.Encrypt(targetKey, nex.NewByteStreamOut(server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	ticket := nex.NewKerberosTicket()
	ticket.SessionKey = sessionKey
	ticket.TargetPID = target.PID
	ticket.InternalData = types.NewBuffer(encryptedTicketInternalData)

	encryptedTicket, err := ticket.Encrypt(sourceKey, nex.NewByteStreamOut(server))
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return []byte{}, nex.Errors.Authentication.Unknown
	}

	return encryptedTicket, 0
}
