package database

import (
	"github.com/PretendoNetwork/nex-go/v2/types"
)

func getSearchDateFilter(dateTime types.DateTime) any {
	if dateTime.IsNever() || dateTime.IsInvalidDateTime() {
		return nil
	}

	return dateTime.Standard()
}
