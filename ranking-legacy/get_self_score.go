package ranking_legacy

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking-legacy/database"
	rankinglegacy "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
)

func (commonProtocol *CommonProtocol) getSelfScore(err error, packet nex.PacketInterface, callID uint32, uniqueID types.UInt32, category types.UInt32, orderParam rankinglegacytypes.RankingOrderParam, length types.UInt8) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	data, nexErr := database.GetNearbyRankings(commonProtocol.manager, connection.PID(), uniqueID, category, orderParam, length)
	if nexErr != nil {
		commonglobals.Logger.Error(nexErr.Error())
		return nil, nexErr
	}

	retval := types.NewInt16(30) // TODO: result codes are unknown

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)
	data.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = rankinglegacy.ProtocolID
	if endpoint.LibraryVersions().Ranking.GreaterOrEqual("2.0.0") {
		rmcResponse.MethodID = rankinglegacy.MethodGetSelfScore
	} else {
		rmcResponse.MethodID = rankinglegacy.MethodGetSelfScoreNEX1
	}
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
