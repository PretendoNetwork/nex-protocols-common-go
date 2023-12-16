package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func findBySingleID(err error, packet nex.PacketInterface, callID uint32, id uint32) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	server := commonProtocol.server

	session, ok := common_globals.Sessions[id]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	bResult := true
	pGathering := nex.NewDataHolder()
	pGathering.SetTypeName("MatchmakeSession")
	pGathering.SetObjectData(session.GameMatchmakeSession)

	rmcResponseStream := nex.NewStreamOut(server)

	rmcResponseStream.WriteBool(bResult)
	rmcResponseStream.WriteDataHolder(pGathering)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodFindBySingleID
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
