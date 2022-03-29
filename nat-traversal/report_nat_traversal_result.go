package nattraversal

import (
	nex "github.com/PretendoNetwork/nex-go"
)

func reportNATTraversalResult(err error, client *nex.Client, callID uint32, cid uint32, result bool, rtt uint32) {
	// NATTraversalMethodReportNATTraversalResult does not exist yet in nex-protocols-go
	/*
		rmcResponse := nex.NewRMCResponse(nexproto.NATTraversalProtocolID, callID)
		rmcResponse.SetSuccess(nexproto.NATTraversalMethodReportNATTraversalResult, nil)

		rmcResponseBytes := rmcResponse.Bytes()

		var responsePacket nex.PacketInterface

		if server.PrudpVersion() == 0 {
			responsePacket, _ = nex.NewPacketV0(client, nil)
			responsePacket.SetVersion(0)
		} else {
			responsePacket, _ = nex.NewPacketV1(client, nil)
			responsePacket.SetVersion(1)
		}

		responsePacket.SetVersion(1)
		responsePacket.SetSource(0xA1)
		responsePacket.SetDestination(0xAF)
		responsePacket.SetType(nex.DataPacket)
		responsePacket.SetPayload(rmcResponseBytes)

		responsePacket.AddFlag(nex.FlagNeedsAck)
		responsePacket.AddFlag(nex.FlagReliable)

		server.Send(responsePacket)
	*/
}
