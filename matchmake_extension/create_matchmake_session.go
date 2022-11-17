package matchmake_extension

import (
	"encoding/hex"
	"fmt"
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func createMatchmakeSession(err error, client *nex.Client, callID uint32, matchmakeSession *nexproto.MatchmakeSession, message string, participationCount uint16) {
	missingHandler := false
	if (commonMatchmakeExtensionProtocol.FindRoomViaMatchmakeSessionHandler == nil){
		logger.Warning("MatchmakeExtension::CreateMatchmakeSession missing FindRoomViaMatchmakeSessionHandler!")
		missingHandler = true
	}
	if (commonMatchmakeExtensionProtocol.AddPlayerToRoomHandler == nil){
		logger.Warning("MatchmakeExtension::CreateMatchmakeSession missing AddPlayerToRoomHandler!")
		missingHandler = true
	}
	if (commonMatchmakeExtensionProtocol.NewRoomHandler == nil){
		logger.Warning("MatchmakeExtension::CreateMatchmakeSession missing NewRoomHandler!")
		missingHandler = true
	}
	if (missingHandler){
		return
	}
	gid := commonMatchmakeExtensionProtocol.NewRoomHandler(client.PID(), client.ConnectionID(), matchmakeSession)
	fmt.Println("===== MATCHMAKE SESSION CREATE =====")
	fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	commonMatchmakeExtensionProtocol.AddPlayerToRoomHandler(gid, client.PID(), client.ConnectionID(), uint32(1))

	matchmakeSession = nexproto.NewMatchmakeSession()

	_, _, matchmakeSession = commonMatchmakeExtensionProtocol.GetRoomHandler(gid)

	rmcResponseStream := nex.NewStreamOut(commonMatchmakeExtensionProtocol.server)
	rmcResponseStream.WriteUInt32LE(matchmakeSession.Gathering.ID)
	rmcResponseStream.WriteBuffer(matchmakeSession.SessionKey)

	rmcResponseBody := rmcResponseStream.Bytes()
	fmt.Println(hex.EncodeToString(rmcResponseBody))

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakeExtensionProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchmakeExtensionMethodCreateMatchmakeSessionWithParam, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if(commonMatchmakeExtensionProtocol.server.PrudpVersion() == 0){
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

	commonMatchmakeExtensionProtocol.server.Send(responsePacket)
	
	rmcMessage := nex.RMCRequest{}
	rmcMessage.SetProtocolID(0xe)
	rmcMessage.SetCallID(0xffff0000+callID)
	rmcMessage.SetMethodID(0x1)
	clientPidString := fmt.Sprintf("%.8x",(client.PID()))
	clientPidString = clientPidString[6:8] + clientPidString[4:6] + clientPidString[2:4] + clientPidString[0:2]
	gidString := fmt.Sprintf("%.8x",(gid))
	gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]
	data, _ := hex.DecodeString("0017000000"+clientPidString+"B90B0000"+gidString+clientPidString+"01000001000000")
	fmt.Println(hex.EncodeToString(data))
	rmcMessage.SetParameters(data)
	rmcMessageBytes := rmcMessage.Bytes()
	
	var messagePacket nex.PacketInterface

	if commonMatchmakeExtensionProtocol.server.PrudpVersion() == 0 {
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

	commonMatchmakeExtensionProtocol.server.Send(messagePacket)
}
