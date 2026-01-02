package ticket_granting

import (
	"encoding/hex"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	ticket_granting "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting"
	ticket_granting_types "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting/types"
)

func (commonProtocol *CommonProtocol) validateAndRequestTicketWithParam(err error, packet nex.PacketInterface, callID uint32) (*nex.RMCMessage, *nex.Error) {
	if err != nil {
		common_globals.Logger.Error(err.Error())
		return nil, nex.NewError(nex.ResultCodes.Core.InvalidArgument, "change_error")
	}

	connection := packet.Sender().(*nex.PRUDPConnection)
	endpoint := connection.Endpoint().(*nex.PRUDPEndPoint)

	parameterStream := nex.NewByteStreamIn(packet.RMCMessage().Parameters, endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	ticketParam := ticket_granting_types.NewValidateAndRequestTicketParam(commonProtocol.protocol.UseCrossplay())
	ticketParam.ExtractFrom(parameterStream)

	if ticketParam.PlatformType != 1 && ticketParam.PlatformType != 2 {
		common_globals.Logger.Errorf("Platform not supported, PID: %s, Platform type: %d", ticketParam.UserName, ticketParam.PlatformType)
		return nil, nex.NewError(nex.ResultCodes.Authentication.InvalidParam, "The requested platform is not supported for this game server.")
	}

	if commonProtocol.protocol.UseCrossplay() {
		if ticketParam.PlatformTypeForPlatformPID != 1 && ticketParam.PlatformTypeForPlatformPID != 2 {
			common_globals.Logger.Errorf("Platform not supported for platform PID, PID: %s, Platform type for platform PID: %d", ticketParam.UserName, ticketParam.PlatformTypeForPlatformPID)
			return nil, nex.NewError(nex.ResultCodes.Authentication.InvalidParam, "The requested platform is not supported for this game server.")
		}
	}

	if ticketParam.ExtraData.Object.DataObjectID().(types.String) != "AuthenticationInfo" {
		common_globals.Logger.Errorf("ExtraData is null or invalid, Object ID: %s", ticketParam.ExtraData.Object.DataObjectID().(types.String))
		return nil, nex.NewError(nex.ResultCodes.Authentication.InvalidParam, "Invalid ExtraData")
	}

	sourceAccount, errorCode := endpoint.AccountDetailsByUsername(string(ticketParam.UserName))

	if errorCode == nil {
		// * The connection doesn't have a PID set here, so we use the source PID
		errorCode = commonProtocol.ValidateLoginData(sourceAccount.PID, ticketParam.ExtraData)
	}

	var targetAccount *nex.Account
	if errorCode == nil {
		targetAccount, errorCode = endpoint.AccountDetailsByUsername(commonProtocol.SecureServerAccount.Username)
	}

	var sourceKey []byte
	if errorCode == nil && sourceAccount.RequiresTokenAuth {
		sourceKey, errorCode = commonProtocol.SourceKeyFromToken(sourceAccount, ticketParam.ExtraData)
	}

	var encryptedTicket []byte
	if errorCode == nil {
		encryptedTicket, errorCode = generateTicket(sourceAccount, targetAccount, sourceKey, commonProtocol.SessionKeyLength, endpoint)
	}

	// * If any errors are triggered, return them
	if errorCode != nil {
		common_globals.Logger.Error(errorCode.Message)
		return nil, errorCode
	}

	ticketResult := ticket_granting_types.NewValidateAndRequestTicketResult(commonProtocol.protocol.UseCrossplay())

	ticketResult.SourcePID = sourceAccount.PID
	ticketResult.BufResponse = types.NewBuffer(encryptedTicket)
	ticketResult.ServiceNodeURL = commonProtocol.SecureStationURL
	ticketResult.CurrentUTCTime.FromTimestamp(time.Now().UTC())
	ticketResult.ReturnMsg = commonProtocol.BuildName

	if endpoint.LibraryVersions().Main.GreaterOrEqual("4.0.0") && sourceKey != nil {
		ticketResult.SourceKey = types.String(hex.EncodeToString(sourceKey))
	}

	if commonProtocol.protocol.UseCrossplay() {
		ticketResult.PlatformPID = sourceAccount.PID
	}

	rmcResponseStream := nex.NewByteStreamOut(endpoint.LibraryVersions(), endpoint.ByteStreamSettings())

	ticketResult.WriteTo(rmcResponseStream)

	rmcResponseBody := rmcResponseStream.Bytes()

	rmcResponse := nex.NewRMCSuccess(endpoint, rmcResponseBody)
	rmcResponse.ProtocolID = ticket_granting.ProtocolID
	rmcResponse.MethodID = ticket_granting.MethodLoginWithContext
	rmcResponse.CallID = callID

	return rmcResponse, nil
}
