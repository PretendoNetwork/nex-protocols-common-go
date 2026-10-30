package ranking_legacy

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	commonglobals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	"github.com/PretendoNetwork/nex-protocols-common-go/v2/ranking-legacy/database"
	rankinglegacy "github.com/PretendoNetwork/nex-protocols-go/v2/ranking/legacy"
)

func (commonProtocol *CommonProtocol) uploadCommonData(err error, packet nex.PacketInterface, callID uint32, uniqueID types.UInt32, commonData types.Buffer) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		commonglobals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	nexErr := database.UploadCommonData(commonProtocol.manager, connection.PID(), uniqueID, commonData)
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
	if endpoint.LibraryVersions().Ranking.GreaterOrEqual("2.0.0") {
		rmcResponse.MethodID = rankinglegacy.MethodUploadCommonData
	} else {
		rmcResponse.MethodID = rankinglegacy.MethodUploadCommonDataNEX1
	}
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
