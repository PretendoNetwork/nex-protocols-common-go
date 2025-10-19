package common_globals

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	account_management_types "github.com/PretendoNetwork/nex-protocols-go/v2/account-management/types"
	ticket_granting_types "github.com/PretendoNetwork/nex-protocols-go/v2/ticket-granting/types"
)

// ValidatePretendoLoginData validates the given Pretendo login data
func ValidatePretendoLoginData(pid types.PID, loginData types.DataHolder, aesKey []byte) *nex.Error {
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

	encryptedToken, err := base64.StdEncoding.DecodeString(tokenBase64)
	if err != nil {
		Logger.Error(err.Error())
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, err.Error())
	}

	decryptedToken, nexError := DecryptToken(encryptedToken, aesKey)
	if nexError != nil {
		return nexError
	}

	// Check for NEX token type
	if decryptedToken.TokenType != 3 {
		return nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Invalid token type")
	}

	// Expire time is in milliseconds
	expireTime := time.Unix(int64(decryptedToken.ExpireTime / 1000), 0)

	if expireTime.Before(time.Now()) {
		return nex.NewError(nex.ResultCodes.Authentication.TokenExpired, "Token expired")
	}

	if types.NewPID(uint64(decryptedToken.UserPID)) != pid {
		return nex.NewError(nex.ResultCodes.Authentication.PrincipalIDUnmatched, fmt.Sprintf("Account %d expected, got %d", pid, decryptedToken.UserPID))
	}

	if decryptedToken.AccessLevel < 0 {
		return nex.NewError(nex.ResultCodes.RendezVous.AccountDisabled, fmt.Sprintf("Account %d is banned", decryptedToken.UserPID))
	}

	return nil
}

// NEXToken is the Pretendo-specific token format
type NEXToken struct {
	SystemType  uint8
	TokenType   uint8
	UserPID     uint32
	ExpireTime  uint64
	TitleID     uint64
	AccessLevel int8
}

// DecryptToken decrypts the given encrypted Pretendo token
func DecryptToken(encryptedToken []byte, aesKey []byte) (*NEXToken, *nex.Error) {
	if len(encryptedToken) < 4 {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Token size is too small")
	}

	// Decrypt the token body
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, err.Error())
	}

	expectedChecksum := binary.BigEndian.Uint32(encryptedToken[0:4])
	encryptedBody := encryptedToken[4:]

	if len(encryptedBody) % aes.BlockSize != 0 {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, fmt.Sprintf("Encrypted body has invalid size %d", len(encryptedBody)))
	}

	decrypted := make([]byte, len(encryptedBody))
	iv := make([]byte, 16)
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(decrypted, encryptedBody)

	paddingSize := int(decrypted[len(decrypted)-1])

	if paddingSize < 0 || paddingSize >= len(decrypted) {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, fmt.Sprintf("Invalid padding size %d for token %x", paddingSize, encryptedToken))
	}

	decrypted = decrypted[:len(decrypted)-paddingSize]

	table := crc32.MakeTable(crc32.IEEE)
	calculatedChecksum := crc32.Checksum(decrypted, table)

	if expectedChecksum != calculatedChecksum {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, "Checksum did not match. Failed decrypt. Are you using the right key?")
	}

	// Unpack the token body to struct
	token := &NEXToken{}
	tokenReader := bytes.NewBuffer(decrypted)

	err = binary.Read(tokenReader, binary.LittleEndian, token)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.Authentication.ValidationFailed, err.Error())
	}

	return token, nil
}
