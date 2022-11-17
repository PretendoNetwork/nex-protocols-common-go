package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
	"fmt"
	"encoding/hex"
)

func updateSessionHost(err error, client *nex.Client, callID uint32, gid uint32) {
	missingHandler := false
	if (UpdateRoomHostHandler == nil){
		logger.Warning("MatchMaking::UpdateSessionHostV1 missing UpdateRoomHostHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	UpdateRoomHostHandler(gid, client.ConnectionID(), client.PID())

	rmcResponse := nex.NewRMCResponse(nexproto.MatchMakingProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchMakingMethodUpdateSessionHost, nil)

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
	
	rmcMessage := nex.RMCRequest{}
	rmcMessage.SetProtocolID(0xe)
	rmcMessage.SetCallID(0xffff0000+callID)
	rmcMessage.SetMethodID(0x1)

	hostpidString := fmt.Sprintf("%.8x",(client.PID()))
	hostpidString = hostpidString[6:8] + hostpidString[4:6] + hostpidString[2:4] + hostpidString[0:2]
	clientPidString := fmt.Sprintf("%.8x",(client.PID()))
	clientPidString = clientPidString[6:8] + clientPidString[4:6] + clientPidString[2:4] + clientPidString[0:2]
	gidString := fmt.Sprintf("%.8x",(gid))
	gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]

	data, _ := hex.DecodeString("0017000000"+hostpidString+"A00F0000"+gidString+clientPidString+"01000001000000")
	rmcMessage.SetParameters(data)
	rmcMessageBytes := rmcMessage.Bytes()

	for _, player := range GetRoomPlayersHandler(gid) {
		if(player[0] == 0 || player[1] == 0){
			continue
		}
	
		targetClient := server.FindClientFromConnectionID(uint32(player[1]))
		if targetClient != nil {

			var messagePacket nex.PacketInterface
		
			if(server.PrudpVersion() == 0){
				messagePacket, _ = nex.NewPacketV0(targetClient, nil)
				messagePacket.SetVersion(0)
			}else{
				messagePacket, _ = nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
			}

			messagePacket.SetSource(0xA1)
			messagePacket.SetDestination(0xAF)
			messagePacket.SetType(nex.DataPacket)
			messagePacket.SetPayload(rmcMessageBytes)

			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)

			server.Send(messagePacket)
		}else{
			logger.Warning("Client not found")
		}
	}

	messagePacket, _ := nex.NewPacketV0(client, nil)
	messagePacket.SetVersion(1)
	messagePacket.SetSource(0xA1)
	messagePacket.SetDestination(0xAF)
	messagePacket.SetType(nex.DataPacket)
	messagePacket.SetPayload(rmcMessageBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	server.Send(messagePacket)
}