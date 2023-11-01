package secureconnection

import (
	"time"

	"github.com/PretendoNetwork/nex-go"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
)

func connect(packet nex.PacketInterface) {
	client := packet.Sender()
	payload := packet.Payload()
	server := commonSecureConnectionProtocol.server

	stream := nex.NewStreamIn(payload, server)

	ticketData, err := stream.ReadBuffer()
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	requestData, err := stream.ReadBuffer()
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	serverKey := nex.DeriveKerberosKey(2, []byte(server.KerberosPassword()))

	ticket := nex.NewKerberosTicketInternalData()
	err = ticket.Decrypt(nex.NewStreamIn(ticketData, server), serverKey)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	ticketTime := ticket.Timestamp().Standard()
	serverTime := time.Now().UTC()

	timeLimit := ticketTime.Add(time.Minute * 2)
	if serverTime.After(timeLimit) {
		common_globals.Logger.Error("Kerberos ticket expired")
		server.TimeoutKick(client)
		return
	}

	sessionKey := ticket.SessionKey()
	kerberos, err := nex.NewKerberosEncryption(sessionKey)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	decryptedRequestData := kerberos.Decrypt(requestData)
	checkDataStream := nex.NewStreamIn(decryptedRequestData, server)

	userPID, err := checkDataStream.ReadUInt32LE()
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	_, err = checkDataStream.ReadUInt32LE() // CID of secure server station url
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	responseCheck, err := checkDataStream.ReadUInt32LE()
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	responseValueStream := nex.NewStreamOut(server)
	responseValueStream.WriteUInt32LE(responseCheck + 1)

	responseValueBufferStream := nex.NewStreamOut(server)
	responseValueBufferStream.WriteBuffer(responseValueStream.Bytes())

	server.AcknowledgePacket(packet, responseValueBufferStream.Bytes())

	err = client.UpdateRC4Key(sessionKey)
	if err != nil {
		common_globals.Logger.Error(err.Error())
		server.TimeoutKick(client)
		return
	}

	client.SetSessionKey(sessionKey)

	client.SetPID(userPID)
}
