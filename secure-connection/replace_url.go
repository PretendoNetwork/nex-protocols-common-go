package secureconnection

import (
	"fmt"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func replaceURL(err error, client *nex.Client, callID uint32, oldStation *nex.StationURL, newStation *nex.StationURL) {
	missingHandler := false
	if (ReplaceConnectionUrlHandler == nil){
		fmt.Println("Secure::ReplaceURL missing ReplaceConnectionUrlHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	ReplaceConnectionUrlHandler(client.ConnectionId(), oldStation.EncodeToString(), newStation.EncodeToString())

	rmcResponse := nex.NewRMCResponse(nexproto.SecureProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.SecureMethodReplaceURL, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if(server.PrudpVersion() == 0){
		responsePacket, _ = nex.NewPacketV0(client, nil)
		responsePacket.SetVersion(0)
	}else{
		responsePacket, _ = nex.NewPacketV1(client, nil)
		responsePacket.SetVersion(1)
	}
	
	responsePacket.SetSource(0xA1)
	responsePacket.SetDestination(0xAF)
	responsePacket.SetType(nex.DataPacket)
	responsePacket.SetPayload(rmcResponseBytes)

	responsePacket.AddFlag(nex.FlagNeedsAck)
	responsePacket.AddFlag(nex.FlagReliable)

	server.Send(responsePacket)
}
