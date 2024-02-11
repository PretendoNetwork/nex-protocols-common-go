package matchmake_extension

import (
	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	matchmake_extension "github.com/PretendoNetwork/nex-protocols-go/matchmake-extension"
)

func updateApplicationBuffer(err error, packet nex.PacketInterface, callID uint32, gid *types.PrimitiveU32, applicationBuffer *types.Buffer) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	session, ok := common_globals.Sessions[gid.Value]
	if !ok {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.SessionVoid, "change_error")
	}

	// TODO - Should ANYONE be allowed to do this??

	session.GameMatchmakeSession.ApplicationBuffer = applicationBuffer.Copy().(*types.Buffer)

	rmcResponse := nex.NewRMCSuccess(endpoint, nil)
	rmcResponse.ProtocolID = matchmake_extension.ProtocolID
	rmcResponse.MethodID = matchmake_extension.MethodUpdateApplicationBuffer
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
