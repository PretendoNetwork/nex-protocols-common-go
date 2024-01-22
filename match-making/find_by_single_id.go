package matchmaking

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func findBySingleID(err error, packet nex.PacketInterface, callID uint32, id *types.PrimitiveU32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[id.Value]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	// TODO - This assumes a PRUDP connection. Refactor to support HPP
	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint
	server := endpoint.Server

	bResult := types.NewPrimitiveBool(true)
	pGathering := types.NewAnyDataHolder()

	pGathering.TypeName = types.NewString("MatchmakeSession")
	pGathering.ObjectData = session.GameMatchmakeSession.Copy()

	rmcResponseStream := nex.NewByteStreamOut(server)

	bResult.WriteTo(rmcResponseStream)
	pGathering.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodFindBySingleID
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
