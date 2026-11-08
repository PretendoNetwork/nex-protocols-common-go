package messaging

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"

	messaging "github.com/PretendoNetwork/nex-protocols-go/v2/messaging"
)

func (commonProtocol *CommonProtocol) deliverMessage(err error, packet nex.PacketInterface, callID uint32, oUserMessage types.DataHolder) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, err.Error())
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	recipientID, recipientType, nexError := commonProtocol.manager.ValidateMessage(oUserMessage)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	oUserMessage, nexError = commonProtocol.manager.PrepareMessage(connection.PID(), oUserMessage)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	oModifiedMessage, lstSandboxNodeID, lstParticipants, nexError := commonProtocol.manager.ProcessMessage(commonProtocol.manager, oUserMessage, types.List[types.UInt64]{recipientID}, recipientType, false)
	if nexError != nil {
		common_globals.Logger.Error(nexError.Error())
		return nil, nexError
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	oModifiedMessage.WriteTo(rmcResponseStream)
	lstSandboxNodeID.WriteTo(rmcResponseStream)
	lstParticipants.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = messaging.ProtocolID
	rmcResponse.MethodID = messaging.MethodDeliverMessage
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
