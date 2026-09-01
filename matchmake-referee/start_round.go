package matchmake_referee

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	database "github.com/PretendoNetwork/nex-protocols-common-go/v2/matchmake-referee/database"
	matchmake_referee "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee"
	matchmake_referee_types "github.com/PretendoNetwork/nex-protocols-go/v2/matchmake-referee/types"
	notifications_constants "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/constants"
	notifications_types "github.com/PretendoNetwork/nex-protocols-go/v2/notifications/types"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

func (commonProtocol *CommonProtocol) StartRound(err error, packet nex.PacketInterface, callID uint32, param matchmake_referee_types.MatchmakeRefereeStartRoundParam) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	if len(param.PIDs) == 0 {
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "Client returned empty list")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	var roundId types.UInt64

	roundId, nexError := database.StartRound(commonProtocol.manager, param)
	if nexError != nil {
		return nil, nexError
	}

	oEvent := notifications_types.NewNotificationEvent()
	oEvent.PIDSource = connection.PID().Copy().(types.PID)
	oEvent.Type = notifications_constants.NotificationCategoryRoundStarted.Build()
	oEvent.Param1 = roundId

	pids := make([]uint64, 0, len(param.PIDs))
	for _, pid := range param.PIDs {
		pids = append(pids, uint64(pid))
	}

	common_globals.SendNotificationEvent(endpoint, oEvent, pids)

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	roundId.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = matchmake_referee.ProtocolID
	rmcResponse.MethodID = matchmake_referee.MethodStartRound
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
