package matchmake_referee

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-referee/database"
	matchmake_referee "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) GetOrCreateStats(err error, packet nex.PacketInterface, callID uint32, param matchmake_referee_types.MatchmakeRefereeStatsInitParam) (*nex.RMCMessage, *nex.Error) {
	// Todo: Potential limit for initialRatingValue?

	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	stats, nexError := matchmake_referee_database.GetOrCreateStats(commonProtocol.manager, connection.PID(), param.Category, param.InitialRatingValue)
	if nexError != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	stats.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_referee.ProtocolID
	rmcResponse.MethodID = matchmake_referee.MethodGetOrCreateStats
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
