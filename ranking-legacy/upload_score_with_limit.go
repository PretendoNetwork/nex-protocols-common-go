package ranking_legacy

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking-legacy/database"
	rankinglegacy "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy"
	rankinglegacytypes "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy/types"
)

// INCOMPLETE: "limit" is unknown
func (commonProtocol *CommonProtocol) uploadScoreWithLimit(err error, packet nex.PacketInterface, callID uint32, uniqueID types.UInt32, category types.UInt32, scores types.List[types.UInt32], unknown1 types.UInt8, unknown2 types.UInt32, limit types.UInt16) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	commonglobals.Logger.Warningf("Limit parameter is unknown and ignored. Ranking accuracy may be affected.")
	// Discard limit for now - we don't know what it does
	scoreList := types.List[rankinglegacytypes.RankingScore]{rankinglegacytypes.RankingScore{
		Category: category,
		Score:    scores,
		Unknown1: unknown1,
		Unknown2: unknown2,
	}}

	nexErr := database.UploadScores(commonProtocol.manager, uniqueID, scoreList, connection.PID())
	if nexErr != nil {
		commonglobals.Logger.Error(nexErr.Error())
		return nil, nexErr
	}

	retval := types.NewInt16(50) // TODO: result codes are unknown

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	retval.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = rankinglegacy.ProtocolID
	rmcResponse.MethodID = rankinglegacy.MethodUploadScoreWithLimit
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
