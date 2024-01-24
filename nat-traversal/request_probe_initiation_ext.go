package nattraversal

import (
	"strconv"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	nat_traversal "github.com/PretendoNetwork/nex-protocols-go/nat-traversal"
)

func requestProbeInitiationExt(err error, packet nex.PacketInterface, callID uint32, targetList *types.List[*types.String], stationToProbe *types.String) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.ResultCodesCore.InvalidArgument
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	rmcResponse := nex.NewRMCSuccess(server, nil)
	rmcResponse.ProtocolID = nat_traversal.ProtocolID
	rmcResponse.MethodID = nat_traversal.MethodRequestProbeInitiationExt
	rmcResponse.CallID = callID

	rmcRequestStream := nex.NewByteStreamOut(server)

	stationToProbe.WriteTo(rmcRequestStream)

	rmcRequestBody := rmcRequestStream.Bytes()

	rmcRequest := nex.NewRMCRequest(server)
	rmcRequest.ProtocolID = nat_traversal.ProtocolID
	rmcRequest.CallID = 0xFFFF0000 + callID
	rmcRequest.MethodID = nat_traversal.MethodInitiateProbe
	rmcRequest.Parameters = rmcRequestBody

	rmcRequestBytes := rmcRequest.Bytes()

	for _, target := range targetList.Slice() {
		targetStation := types.NewStationURL(target.Value)

		if connectionIDString, ok := targetStation.Fields["RVCID"]; ok {
			connectionID, err := strconv.Atoi(connectionIDString)
			if err != nil {
				common_globals.Logger.Error(err.Error())
			}

			target := endpoint.FindConnectionByID(uint32(connectionID))
			if target == nil {
				common_globals.Logger.Warning("Client not found")
				continue
			}

			var messagePacket nex.PRUDPPacketInterface

			if target.DefaultPRUDPVersion == 0 {
				messagePacket, _ = nex.NewPRUDPPacketV0(target, nil)
			} else {
				messagePacket, _ = nex.NewPRUDPPacketV1(target, nil)
			}

			messagePacket.SetType(nex.DataPacket)
			messagePacket.AddFlag(nex.FlagNeedsAck)
			messagePacket.AddFlag(nex.FlagReliable)
			messagePacket.SetSourceVirtualPortStreamType(target.StreamType)
			messagePacket.SetSourceVirtualPortStreamID(target.Endpoint.StreamID)
			messagePacket.SetDestinationVirtualPortStreamType(target.StreamType)
			messagePacket.SetDestinationVirtualPortStreamID(target.StreamID)
			messagePacket.SetPayload(rmcRequestBytes)

			server.Send(messagePacket)
		}
	}

	return rmcResponse, 0
}
