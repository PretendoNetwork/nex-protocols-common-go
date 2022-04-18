package secureconnection

import (
	"github.com/PretendoNetwork/nex-go"
)

func connect(packet nex.PacketInterface) {
	payload := packet.Payload()
	server := commonSecureConnectionProtocol.server

	stream := nex.NewStreamIn(payload, server)

	// TODO: Error check!!
	ticketData, _ := stream.ReadBuffer()
	requestData, _ := stream.ReadBuffer()

	serverKey := nex.DeriveKerberosKey(2, []byte(server.KerberosPassword()))

	ticket := nex.NewKerberosTicketInternalData()
	ticket.Decrypt(nex.NewStreamIn(ticketData, server), serverKey)

	// TODO: Check timestamp here

	sessionKey := ticket.SessionKey()
	kerberos := nex.NewKerberosEncryption(sessionKey)

	decryptedRequestData := kerberos.Decrypt(requestData)
	checkDataStream := nex.NewStreamIn(decryptedRequestData, server)

	userPID := checkDataStream.ReadUInt32LE()
	_ = checkDataStream.ReadUInt32LE() //CID of secure server station url
	responseCheck := checkDataStream.ReadUInt32LE()

	responseValueStream := nex.NewStreamOut(server)
	responseValueStream.WriteUInt32LE(responseCheck + 1)

	responseValueBufferStream := nex.NewStreamOut(server)
	responseValueBufferStream.WriteBuffer(responseValueStream.Bytes())

	server.AcknowledgePacket(packet, responseValueBufferStream.Bytes())

	packet.Sender().UpdateRC4Key(sessionKey)
	packet.Sender().SetSessionKey(sessionKey)

	packet.Sender().SetPID(userPID)
}
