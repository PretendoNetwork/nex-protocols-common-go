package common_globals

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	pb "github.com/PretendoNetwork/grpc/go/account/v2"
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	account_management_types "github.com/PretendoNetwork/nex-protocols-go/v2/account-management/types"
	ticket_granting_types "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func ConnectToAccountGRPC(host string, port uint16, apiKey string) {
	if GRPCAccountClientConnection != nil {
		return
	}

	var err error

	GRPCAccountClientConnection, err = grpc.NewClient(fmt.Sprintf("%s:%d", host, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		Logger.Criticalf("Failed to connect to account gRPC server: %v", err)
		os.Exit(0)
	}

	GRPCAccountClient = pb.NewAccountServiceClient(GRPCAccountClientConnection)
	GRPCAccountCommonMetadata = metadata.Pairs(
		"X-API-Key", apiKey,
	)
}

// ValidatePNLoginData validates the given NEX login data
func ValidatePNLoginData(pid types.PID, loginData types.DataHolder, gameServerID string) *nex.Error {
	var tokenBase64 string

	loginDataType := loginData.Object.DataObjectID().(types.String)

	switch loginDataType {
	case "NintendoLoginData": // * Wii U
		nintendoCreateAccountData := loginData.Object.Copy().(ticket_granting_types.NintendoLoginData)

		tokenBase64 = string(nintendoCreateAccountData.Token)
	case "AccountExtraInfo": // * 3DS
		accountExtraInfo := loginData.Object.Copy().(account_management_types.AccountExtraInfo)

		tokenBase64 = string(accountExtraInfo.NEXToken)
		tokenBase64 = strings.Replace(tokenBase64, ".", "+", -1)
		tokenBase64 = strings.Replace(tokenBase64, "-", "/", -1)
		tokenBase64 = strings.Replace(tokenBase64, "*", "=", -1)
	case "AuthenticationInfo": // * 3DS / Wii U
		authenticationInfo := loginData.Object.Copy().(ticket_granting_types.AuthenticationInfo)

		tokenBase64 = string(authenticationInfo.Token)
		tokenBase64 = strings.Replace(tokenBase64, ".", "+", -1)
		tokenBase64 = strings.Replace(tokenBase64, "-", "/", -1)
		tokenBase64 = strings.Replace(tokenBase64, "*", "=", -1)
	default:
		Logger.Errorf("Invalid loginData data type %s!", loginDataType)
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, fmt.Sprintf("Invalid loginData data type %s!", loginDataType))
	}

	ctx := metadata.NewOutgoingContext(context.Background(), GRPCAccountCommonMetadata)

	response, err := GRPCAccountClient.ExchangeNEXTokenForUserData(ctx, &pb.ExchangeNEXTokenForUserDataRequest{
		GameServerId: gameServerID,
		Token:        tokenBase64,
	})
	if err != nil {
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, err.Error())
	}

	// * The account server database separates all the token types into their own
	// * collections, so a non-NEX token (even if valid) should still return no
	// * data here. But sanity the types check anyway just in case
	if response.TokenInfo.TokenType != 3 { // * 3 = NEX
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Invalid token")
	}

	if response.TokenInfo.SystemType != 1 && response.TokenInfo.SystemType != 2 { // * 1 = WUP, 2 = CTR
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Invalid token")
	}

	// * If the token is expired, the account server database will have deleted it,
	// * but sanity check anyway just in case
	if response.TokenInfo.ExpireTime != nil && response.TokenInfo.ExpireTime.AsTime().Before(time.Now()) {
		return nex.NewError(nex.ResultCodes.Authentication.TokenExpired, "Token expired")
	}

	if types.NewPID(uint64(response.NexAccount.Pid)) != pid {
		return nex.NewError(nex.ResultCodes.Authentication.PrincipalIDUnmatched, fmt.Sprintf("Account %d expected, got %d", pid, response.NexAccount.Pid))
	}

	if response.NexAccount.AccessLevel < 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.AccountDisabled, fmt.Sprintf("Account %d is banned", response.NexAccount.Pid))
	}

	return nil
}
