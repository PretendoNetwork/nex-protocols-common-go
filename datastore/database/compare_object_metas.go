package database

import (
	"slices"

	"github.com/PretendoNetwork/nex-go/v2"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
)

func compareObjectMetas(currentData datastore_types.DataStoreMetaInfo, compareParam datastore_types.DataStoreChangeMetaCompareParam) *nex.Error {
	compareName := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagName)
	compareAccessPermission := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagAccessPermission)
	compareUpdatePermission := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagUpdatePermission)
	comparePeriod := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagPeriod)
	compareMetaBinary := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagMetaBinary)
	compareTags := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagTags)
	compareDataType := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagDataType)
	compareReferredCount := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagReferredCount)
	compareStatus := compareParam.ComparisonFlag.HasFlag(datastore_constants.ComparisonFlagStatus)

	if compareParam.ComparisonFlag == datastore_constants.ComparisonFlagAll {
		compareName = true
		compareAccessPermission = true
		compareUpdatePermission = true
		comparePeriod = true
		compareMetaBinary = true
		compareTags = true
		compareDataType = true
		compareReferredCount = true
		compareStatus = true
	}

	if compareName && !currentData.Name.Equals(compareParam.Name) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareAccessPermission && !currentData.Permission.Equals(compareParam.Permission) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareUpdatePermission && !currentData.DelPermission.Equals(compareParam.DelPermission) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if comparePeriod && !currentData.Period.Equals(compareParam.Period) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareMetaBinary && !currentData.MetaBinary.Equals(compareParam.MetaBinary) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareTags {
		currentTags := slices.Sorted(slices.Values(currentData.Tags))
		compareTags := slices.Sorted(slices.Values(compareParam.Tags))

		if !slices.Equal(currentTags, compareTags) {
			return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
		}
	}

	if compareDataType && !currentData.DataType.Equals(compareParam.DataType) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareReferredCount && !currentData.ReferredCnt.Equals(compareParam.ReferredCnt) {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	if compareStatus && currentData.Status != compareParam.Status {
		return nex.NewError(nex.ResultCodes.DataStore.ValueNotEqual, "change_error")
	}

	return nil
}
