package matchmake_extension

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"math"

	nex "github.com/PretendoNetwork/nex-go"
	nexproto "github.com/PretendoNetwork/nex-protocols-go"
)

func autoMatchmake_Postpone(err error, client *nex.Client, callID uint32, matchmakeSession *nexproto.MatchmakeSession, message string) {	
	gid := FindRoomViaMatchmakeSessionHandler(matchmakeSession)
	if gid == math.MaxUint32 {
		gid = NewRoomHandler(client.PID(), matchmakeSession)
	}

	fmt.Println("GATHERING ID: " + strconv.Itoa((int)(gid)))

	AddPlayerToRoomHandler(gid, client.PID(), uint32(1))

	hostpid, matchmakeSession := GetRoomHandler(gid)

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteString("MatchmakeSession")
	lengthStream := nex.NewStreamOut(server)
	lengthStream.WriteStructure(matchmakeSession.Gathering)
	lengthStream.WriteStructure(matchmakeSession)
	matchmakeSessionLength := uint32(len(lengthStream.Bytes()))
	rmcResponseStream.WriteUInt32LE(matchmakeSessionLength + 4)
	rmcResponseStream.WriteUInt32LE(matchmakeSessionLength)
	rmcResponseStream.WriteStructure(matchmakeSession.Gathering)
	rmcResponseStream.WriteStructure(matchmakeSession)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCResponse(nexproto.MatchmakeExtensionProtocolID, callID)
	rmcResponse.SetSuccess(nexproto.MatchmakeExtensionMethodAutoMatchmake_Postpone, rmcResponseBody)

	rmcResponseBytes := rmcResponse.Bytes()
	
	var responsePacket nex.PacketInterface

	if server.PrudpVersion() == 0 {
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

	server.Send(responsePacket)
	
	rmcMessage := nex.RMCRequest{}
	rmcMessage.SetProtocolID(0xe)
	rmcMessage.SetCallID(0xffff0000+callID)
	rmcMessage.SetMethodID(0x1)
	clientPidString := fmt.Sprintf("%.8x",(client.PID()))
	clientPidString = clientPidString[6:8] + clientPidString[4:6] + clientPidString[2:4] + clientPidString[0:2]
	gidString := fmt.Sprintf("%.8x",(gid))
	gidString = gidString[6:8] + gidString[4:6] + gidString[2:4] + gidString[0:2]
	data, _ = hex.DecodeString("0017000000"+hostpidString+"B90B0000"+gidString+clientPidString+"01000001000000")
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
