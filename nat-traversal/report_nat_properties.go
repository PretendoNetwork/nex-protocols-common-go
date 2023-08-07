package nattraversal

import (
	"strconv"

	nex "github.com/PretendoNetwork/nex-go"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func reportNATProperties(err error, client *nex.Client, callID uint32, natm uint32, natf uint32, rtt uint32) uint32 {
	if err != nil {
		logger.Error(err.Error())
		return nex.Errors.Core.InvalidArgument
	}

	server := commonNATTraversalProtocol.server

	stationURLsStrings := client.StationURLs()
	stationURLs := make([]*nex.StationURL, len(stationURLsStrings))

	for i := 0; i < len(stationURLsStrings); i++ {
		stationURLs[i] = nex.NewStationURL(stationURLsStrings[i])
		if stationURLs[i].Type() == "3" {
			natm_s := strconv.FormatUint(uint64(natm), 10)
			natf_s := strconv.FormatUint(uint64(natf), 10)
			stationURLs[i].SetNatm(natm_s)
			stationURLs[i].SetNatf(natf_s)
		}
		stationURLsStrings[i] = stationURLs[i].EncodeToString()
	}

	client.SetStationURLs(stationURLsStrings)

	rmcResponse := nex.NewRMCResponse(nat_traversal.ProtocolID, callID)
	rmcResponse.SetSuccess(nat_traversal.MethodReportNATProperties, nil)

	rmcResponseBytes := rmcResponse.Bytes()

	var responsePacket nex.PacketInterface

	if server.PRUDPVersion() == 0 {
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

	return 0
}
