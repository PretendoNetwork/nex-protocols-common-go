package matchmake_extension

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"math"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

var testGid uint32

func autoMatchmakeWithParam_Postpone(err error, client *nex.Client, callID uint32, matchmakeSession *nexproto.MatchmakeSession, sourceGid uint32) {
	missingHandler := false
	if (FindRoomViaMatchmakeSessionHandler == nil){
		logger.Warning("MatchmakeExtension::AutoMatchmakeWithParam_Postpone missing FindRoomViaMatchmakeSessionHandler!")
		missingHandler = true
	}
	if (AddPlayerToRoomHandler == nil){
		logger.Warning("MatchmakeExtension::AutoMatchmakeWithParam_Postpone missing AddPlayerToRoomHandler!")
		missingHandler = true
	}
	if (NewRoomHandler == nil){
		logger.Warning("MatchmakeExtension::AutoMatchmakeWithParam_Postpone missing NewRoomHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	var gid uint32

	//Splatfest code, there's gotta be a better way to handle this.
	/*fmt.Println(sourceGid)
	
	if((int)(matchmakeSession.GameMode) == 12){
		var team uint32
		if(matchmakeSession.Attributes[3] == 0){
			team = 1
		}else{
			team = 0
		}
		gid = findRoom(matchmakeSession.GameMode, true, team, matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	}else{
		gid = findRoom(matchmakeSession.GameMode, true, matchmakeSession.Attributes[3], matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	}
	if gid == math.MaxUint32 {
		gid = newRoom(client.PID(), matchmakeSession.GameMode, true, matchmakeSession.Attributes[3], matchmakeSession.Attributes[2], uint32(1), matchmakeSession.Attributes[5]&0xF)
	}

	if((int)(matchmakeSession.GameMode) == 12){
		rmcMessage := nex.RMCRequest{}
		rmcMessage.SetProtocolID(0xe)
		rmcMessage.SetCallID(0xffff0000+callID)
		rmcMessage.SetMethodID(0x1)
	
		hostpidString := fmt.Sprintf("%.8x",(client.PID()))
		hostpidString = hostpidString[6:8] + hostpidString[4:6] + hostpidString[2:4] + hostpidString[0:2]
		gidString := fmt.Sprintf("%.8x",(gid))
		gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]
	
		for _, pid := range getRoomPlayers(sourceGid) {
			if(pid == 0){
				continue
			}
			targetClient := nexServer.FindClientFromPID(uint32(pid))
			if targetClient != nil {
				clientPidString := fmt.Sprintf("%.8x",(pid))
				clientPidString = clientPidString[6:8] + clientPidString[4:6] + clientPidString[2:4] + clientPidString[0:2]
	
				data, _ := hex.DecodeString("0017000000"+hostpidString+"90DC0100"+gidString+clientPidString+"01000001000000")
				rmcMessage.SetParameters(data)
				rmcMessageBytes := rmcMessage.Bytes()
				messagePacket, _ := nex.NewPacketV1(targetClient, nil)
				messagePacket.SetVersion(1)
				messagePacket.SetSource(0xA1)
				messagePacket.SetDestination(0xAF)
				messagePacket.SetType(nex.DataPacket)
				messagePacket.SetPayload(rmcMessageBytes)
	
				messagePacket.AddFlag(nex.FlagNeedsAck)
				messagePacket.AddFlag(nex.FlagReliable)
	
				nexServer.Send(messagePacket)
			}else{
				fmt.Println("not found")
			}
		}
	}*/

	//fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	gid = FindRoomViaMatchmakeSessionHandler(matchmakeSession)
	if gid == math.MaxUint32 {
		gid = NewRoomHandler(client.PID(), matchmakeSession)
	}

	fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	AddPlayerToRoomHandler(gid, client.PID(), uint32(1))

	hostpid, matchmakeSession := GetRoomHandler(gid)

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteStructure(matchmakeSession.Gathering)
	rmcResponseStream.WriteStructure(matchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()
	fmt.Println(hex.EncodeToString(rmcResponseBody))
	hostpidString := fmt.Sprintf("%.8x",(hostpid))
	hostpidString = hostpidString[6:8] + hostpidString[4:6] + hostpidString[2:4] + hostpidString[0:2]
	clientPidString := fmt.Sprintf("%.8x",(client.PID()))
	clientPidString = clientPidString[6:8] + clientPidString[4:6] + clientPidString[2:4] + clientPidString[0:2]
	gidString := fmt.Sprintf("%.8x",(gid))
	gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]
	data, _ := hex.DecodeString("0023000000"+gidString+hostpidString+hostpidString+"000008005f00000000000000000a000000000000010000035c01000001000000060000008108020107000000020000000100000010000000000000000101000000d4000000088100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ea801c8b0000000000010100410000000010011c010000006420000000161466a08c8df18b118ed5a67650a47435f081d09804a7c1902b145e18eff47c00000000001c000000020000000400405352000301050040474952000103000000000000008f7e9e961f000000010000000000000000")

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakeExtensionProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchmakeExtensionMethodAutoMatchmakeWithParam_Postpone, rmcResponseBody)

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
	if(matchmakeSession.GameMode == 12){
		//gidString := fmt.Sprintf("%.8x",(testGid))
		//gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]
		data, _ = hex.DecodeString("0017000000"+hostpidString+"B90B0000"+gidString+clientPidString+"01000004000000")
	}else{
		data, _ = hex.DecodeString("0017000000"+hostpidString+"B90B0000"+gidString+clientPidString+"01000001000000")
		matchmakeSession.GameMode = 2 
	}
	fmt.Println(hex.EncodeToString(data))
	rmcMessage.SetParameters(data)
	rmcMessageBytes := rmcMessage.Bytes()
	
	targetClient := server.FindClientFromPID(uint32(hostpid))
	
	var messagePacket nex.PacketInterface

	if server.PrudpVersion() == 0 {
		messagePacket, _ = nex.NewPacketV0(client, nil)
		messagePacket.SetVersion(0)
	} else {
		messagePacket, _ = nex.NewPacketV1(client, nil)
		messagePacket.SetVersion(1)
	}
	messagePacket.SetSource(0xA1)
	messagePacket.SetDestination(0xAF)
	messagePacket.SetType(nex.DataPacket)
	messagePacket.SetPayload(rmcMessageBytes)

	messagePacket.AddFlag(nex.FlagNeedsAck)
	messagePacket.AddFlag(nex.FlagReliable)

	server.Send(messagePacket)
	
	if server.PrudpVersion() == 0 {
		messagePacket, _ = nex.NewPacketV0(targetClient, nil)
		messagePacket.SetVersion(0)
	} else {
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
}
