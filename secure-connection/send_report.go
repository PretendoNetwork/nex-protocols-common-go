package secureconnection

import (
	"encoding/hex"
	"fmt"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	secure_connection "github.com/PretendoNetwork/nex-protocols-go/secure-connection"
)

func sendReport(err error, client *nex.Client, callID uint32, reportID uint32, reportData []byte) {
	fmt.Println("Report ID: " + strconv.Itoa(int(reportID)))
	fmt.Println("Report Data: " + hex.EncodeToString(reportData))

	rmcResponse := nex.NewRMCResponse(secure_connection.ProtocolID, callID)
	rmcResponse.SetSuccess(secure_connection.MethodSendReport, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if commonSecureConnectionProtocol.server.PRUDPVersion() == 0 {
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	} else {
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}

	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	commonSecureConnectionProtocol.server.Send(responsePacket)
}
