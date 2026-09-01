package matchmake_referee

import (
	"github.com/PretendoNetwork/nex-go/v2"
	database "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-referee/database"
	matchmake_referee "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) EndRound(err error, packet nex.PacketInterface, callID uint32, param matchmake_referee_types.MatchmakeRefereeEndRoundParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	nexError := database.EndRound(commonProtocol.manager, connection.PID(), param.RoundID, param.PersonalRoundResults)
	if nexError != nil {
		return nil, nexError
	}

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_referee.ProtocolID
	rmcResponse.MethodID = matchmake_referee.MethodEndRound
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
