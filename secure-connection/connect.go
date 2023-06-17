package secureconnection

import (
	"time"
	"github.com/PretendoNetwork/nex-go"
)

func connect(packet nex.PacketInterface) {
	payload := packet.Payload()
	server := commonSecureConnectionProtocol.server

	stream := nex.NewStreamIn(payload, server)

	ticketData, err := stream.ReadBuffer()
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	requestData, err := stream.ReadBuffer()
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	serverKey := nex.DeriveKerberosKey(2, []byte(server.KerberosPassword()))

	ticket := nex.NewKerberosTicketInternalData()
	err = ticket.Decrypt(nex.NewStreamIn(ticketData, server), serverKey)
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	ticketTime := ticket.Timestamp().Standard()
	serverTime := time.Now().UTC()

	timeLimit := ticketTime.Add(time.Minute * 2)
	if serverTime.After(timeLimit) {
		logger.Error("Kerberos ticket expired")
		server.TimeoutKick(packet.Sender())
		return
	}

	sessionKey := ticket.SessionKey()
	kerberos, err := nex.NewKerberosEncryption(sessionKey)
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	decryptedRequestData := kerberos.Decrypt(requestData)
	checkDataStream := nex.NewStreamIn(decryptedRequestData, server)

	userPID, err := checkDataStream.ReadUInt32LE()
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	_, err = checkDataStream.ReadUInt32LE() // CID of secure server station url
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	responseCheck, err := checkDataStream.ReadUInt32LE()
	if err != nil {
		logger.Error(err.Error())
		server.TimeoutKick(packet.Sender())
		return
	}

	responseValueStream := nex.NewStreamOut(server)
	responseValueStream.WriteUInt32LE(responseCheck + 1)

	responseValueBufferStream := nex.NewStreamOut(server)
	responseValueBufferStream.WriteBuffer(responseValueStream.Bytes())

	server.AcknowledgePacket(packet, responseValueBufferStream.Bytes())

	packet.Sender().UpdateRC4Key(sessionKey)
	packet.Sender().SetSessionKey(sessionKey)

	packet.Sender().SetPID(userPID)
}
