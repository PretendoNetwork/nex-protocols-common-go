package matchmaking

import (
	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	match_making "github.com/PretendoNetwork/nex-protocols-go/match-making"
)

func updateSessionURL(err error, packet nex.PacketInterface, callID uint32, idGathering uint32, strURL string) (*nex.RMCMessage, uint32) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.Errors.Core.InvalidArgument
	}

	session, ok := common_globals.Sessions[idGathering]
	if !ok {
		return nil, nex.Errors.RendezVous.SessionVoid
	}

	server := commonProtocol.server
	client := packet.Sender()

	// * Mario Kart 7 seems to set an empty strURL, so I assume this is what the method does?
	session.GameMatchmakeSession.Gathering.HostPID = client.PID()
	if session.GameMatchmakeSession.Gathering.Flags&match_making.GatheringFlags.DisconnectChangeOwner != 0 {
		session.GameMatchmakeSession.Gathering.OwnerPID = client.PID()
	}

	rmcResponseStream := nex.NewStreamOut(server)
	rmcResponseStream.WriteBool(true) // * %retval%

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(server, rmcResponseBody)
	rmcResponse.ProtocolID = match_making.ProtocolID
	rmcResponse.MethodID = match_making.MethodGetSessionURLs
	rmcResponse.CallID = callID

	return rmcResponse, 0
}
