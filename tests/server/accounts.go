package main

import (
	"slices"
	"strconv"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
)

const (
	testUserPIDMin   uint64 = 1000
	testUserPIDMax   uint64 = 1999
	testUserPassword        = "password"
)

var authenticationServerAccount = nex.NewAccount(types.NewPID(1), "Quazal Authentication", "authpassword", false)
var secureServerAccount = nex.NewAccount(types.NewPID(2), "Quazal Rendez-Vous", "securepassword", false)

// * Bypass the friends server, fuck it we ball
var friends = map[uint32][]uint32{
	1000: {1001},
	1001: {1000},
}

func accountDetailsByPID(pid types.PID) (*nex.Account, *nex.Error) {
	if pid.Equals(authenticationServerAccount.PID) {
		return authenticationServerAccount, nil
	}

	if pid.Equals(secureServerAccount.PID) {
		return secureServerAccount, nil
	}

	// * Allow any PID between 1000 and 1999 to login with the same password,
	// * to simulate multiple users without needing to set them all up first
	if uint64(pid) >= testUserPIDMin && uint64(pid) <= testUserPIDMax {
		return nex.NewAccount(pid, strconv.FormatUint(uint64(pid), 10), testUserPassword, false), nil
	}

	return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidPID, "Invalid PID")
}

func accountDetailsByUsername(username string) (*nex.Account, *nex.Error) {
	if username == authenticationServerAccount.Username {
		return authenticationServerAccount, nil
	}

	if username == secureServerAccount.Username {
		return secureServerAccount, nil
	}

	pid, err := strconv.ParseUint(username, 10, 64)
	if err != nil {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidUsername, "Invalid username")
	}

	account, errCode := accountDetailsByPID(types.NewPID(pid))
	if errCode != nil {
		return nil, nex.NewError(nex.ResultCodes.RendezVous.InvalidUsername, "Invalid username")
	}

	return account, nil
}

func getUserFriendPIDs(pid uint32) []uint32 {
	return slices.Clone(friends[pid])
}
