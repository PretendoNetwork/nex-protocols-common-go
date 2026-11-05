package database

import (
	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/lib/pq"
)

func SearchObjectsByDataStoreSearchParam(database common_globals.Database, manager *common_globals.DataStoreManager, callerPID types.PID, param datastore_types.DataStoreSearchParam, totalCountEnabled bool) (datastore_types.DataStoreSearchResult, *nex.Error) {
	pSearchResult := datastore_types.NewDataStoreSearchResult()

	errCode := validateSearchParam(callerPID, param)
	if errCode != nil {
		return pSearchResult, errCode
	}

	friendPIDs := make([]uint32, 0)
	if manager.GetUserFriendPIDs != nil {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friendPIDs = append(friendPIDs, manager.GetUserFriendPIDs(uint32(callerPID))...)
	}

	dataTypes := make([]types.UInt16, 0, len(param.DataTypes)+1)
	if param.DataType != types.UInt16(datastore_constants.InvalidDataType) {
		dataTypes = append(dataTypes, param.DataType)
	}

	dataTypes = append(dataTypes, param.DataTypes...)

	// * Only a certain number of results are actually counted
	useAnyOffset := uint64(param.ResultRange.Offset) == datastore_constants.ResultRangeAnyOffset
	offset := uint64(param.ResultRange.Offset)
	countedLimit := max(1000, offset+5*uint64(param.ResultRange.Length))

	if useAnyOffset {
		offset = 0
		countedLimit = 1000
	}

	var matchedCount uint64
	var dataIDs types.List[types.UInt64]

	err := database.QueryRow(`
		WITH matches AS (
			SELECT
				o.data_id,
				o.name,
				CASE $14::int -- Check which column is used to sort
					WHEN 0 THEN o.data_id
					WHEN 1 THEN v.size
					WHEN 3 THEN o.data_type
					WHEN 4 THEN o.reference_count
					WHEN 5 THEN extract(epoch FROM o.creation_date)
					WHEN 6 THEN extract(epoch FROM o.update_date)
					WHEN 7 THEN o.owner
					ELSE CASE -- Rating/rating average columns
						WHEN $14 < 96 THEN r.total_value
						WHEN r.rating_count = 0 THEN 0
						ELSE (r.total_value - r.initial_value) / r.rating_count
					END
				END AS sort_column
			FROM datastore.objects o
			JOIN datastore.object_versions v ON v.data_id = o.data_id AND v.version = (SELECT MAX(version) FROM datastore.object_versions WHERE data_id = o.data_id AND completed = true)
			LEFT JOIN datastore.rating_slots r ON r.data_id = o.data_id AND r.slot = CASE
				WHEN $14 BETWEEN 64 AND 79 THEN $14 - 64 -- Ratings, total
				WHEN $14 BETWEEN 96 AND 111 THEN $14 - 96 -- Ratings, average
			END
			WHERE o.deleted = false AND o.expiration_date > (CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
			AND CASE $3::int -- Check the SearchTarget
				-- Public objects
				WHEN 1 THEN o.access_permission = 0 AND o.status = 0

				-- Objects owned by the caller and set to PermissionFriend
				WHEN 2 THEN o.owner = $1 AND o.access_permission = 1 AND o.status = 0

				-- Objects owned by the caller and set to PermissionSpecified
				WHEN 3 THEN o.owner = $1 AND o.access_permission = 2 AND o.status = 0

				-- Objects owned by the caller and set to PermissionSpecifiedFriend
				WHEN 4 THEN o.owner = $1 AND o.access_permission = 4 AND o.status = 0

				-- Objects owned by the caller and set to PermissionFriend, PermissionSpecified or PermissionSpecifiedFriend
				WHEN 5 THEN o.owner = $1 AND o.access_permission IN (1, 2, 4) AND o.status = 0

				-- Objects owned by the friends of the caller and set to PermissionFriend
				WHEN 6 THEN o.owner = ANY($2::numeric[]) AND o.access_permission = 1 AND o.status = 0

				-- Objects set to either PermissionSpecified or PermissionSpecifiedFriend and have the caller specified
				WHEN 7 THEN $1 = ANY(o.access_permission_recipients) AND (o.access_permission = 2 OR (o.access_permission = 4 AND o.owner = ANY($2::numeric[]))) AND o.status = 0

				-- Objects either owned by the friends of the caller and set to PermissionFriend, or objects set to either PermissionSpecified or PermissionSpecifiedFriend and have the caller specified
				WHEN 8 THEN (($1 = ANY(o.access_permission_recipients) AND (o.access_permission = 2 OR (o.access_permission = 4 AND o.owner = ANY($2::numeric[])))) OR (o.owner = ANY($2::numeric[]) AND o.access_permission = 1)) AND o.status = 0

				-- Private objects owned by the caller
				WHEN 9 THEN o.owner = $1 AND o.access_permission = 3 AND o.status = 0

				-- Objects owned by the caller with DataStatusNone
				WHEN 10 THEN o.owner = $1 AND o.status = 0

				-- Public objects that belong to anyone besides the owner and their friends
				WHEN 11 THEN o.access_permission = 0 AND o.owner != $1 AND NOT o.owner = ANY($2::numeric[]) AND o.status = 0

				-- Objects owned by the caller with DataStatusPending
				WHEN 12 THEN o.owner = $1 AND o.status = 2

				-- Objects owned by the caller with DataStatusRejected
				WHEN 13 THEN o.owner = $1 AND o.status = 5

				-- Objects owned by the caller with DataStatusNone, DataStatusPending or DataStatusRejected
				WHEN 14 THEN o.owner = $1 AND o.status IN (0, 2, 5)
			END
			AND CASE $4::int -- Check the OwnerType
				-- Objects owned by anyone if OwnerIDs is empty, or only those owned by OwnerIDs
				WHEN 0 THEN cardinality($5::numeric[]) = 0 OR o.owner = ANY($5::numeric[])

				-- Objects owned by friends of the caller
				WHEN 1 THEN o.owner = ANY($2::numeric[])

				-- Objects by anyone except those in OwnerIDs
				WHEN 2 THEN NOT o.owner = ANY($5::numeric[])
			END
			-- Objects which list DestinationIDs as recipients
			AND ($3 IN (7, 8) OR cardinality($6::numeric[]) = 0 OR (o.access_permission IN (2, 4) AND o.access_permission_recipients && $6::numeric[]))

			-- Filter DataTypes
			AND (cardinality($7::int[]) = 0 OR o.data_type = ANY($7::int[]))

			-- Creation/update timestamp filters
			AND ($8::timestamp IS NULL OR date_trunc('second', o.creation_date) >= $8::timestamp)
			AND ($9::timestamp IS NULL OR date_trunc('second', o.creation_date) <= $9::timestamp)
			AND ($10::timestamp IS NULL OR date_trunc('second', o.update_date) >= $10::timestamp)
			AND ($11::timestamp IS NULL OR date_trunc('second', o.update_date) <= $11::timestamp)

			-- Check ReferDataID
			AND ($12 = 0 OR o.refer_data_id = $12)

			-- Check tags. Trailing spaces is ignored
			AND ARRAY(SELECT rtrim(tag, ' ') FROM unnest(o.tags) AS tag) @> ARRAY(SELECT rtrim(tag, ' ') FROM unnest($13::text[]) AS tag)

			-- Drop objects which dont have the rating column, if used
			AND ($14 < 64 OR r.data_id IS NOT NULL)

			-- check MinimalRatingFrequency
			AND COALESCE(r.rating_count, 0) >= $16
		)
		SELECT
			(SELECT count(*) FROM (SELECT 1 FROM matches LIMIT $20::bigint + 1) AS counted),
			ARRAY(
				SELECT data_id
				FROM matches
				ORDER BY
					CASE WHEN $17 THEN random() END, -- If ANY_OFFSET is used, shuffle the results randomly. Name is misleading and we cheat
					CASE WHEN $15 = 0 THEN sort_column END ASC,
					CASE WHEN $15 = 1 THEN sort_column END DESC,
					(CASE WHEN $15 = 0 AND $14 = 2 THEN name END) COLLATE datastore.search_name ASC,
					(CASE WHEN $15 = 1 AND $14 = 2 THEN name END) COLLATE datastore.search_name DESC,
					data_id
				OFFSET $18
				LIMIT $19
			)
	`,
		callerPID,
		pq.Array(friendPIDs),
		param.SearchTarget,
		param.OwnerType,
		pq.Array(param.OwnerIDs),
		pq.Array(param.DestinationIDs),
		pq.Array(dataTypes),
		getSearchDateFilter(param.CreatedAfter),
		getSearchDateFilter(param.CreatedBefore),
		getSearchDateFilter(param.UpdatedAfter),
		getSearchDateFilter(param.UpdatedBefore),
		param.ReferDataID,
		pq.Array(param.Tags),
		param.ResultOrderColumn,
		param.ResultOrder,
		param.MinimalRatingFrequency,
		useAnyOffset,
		offset,
		param.ResultRange.Length,
		countedLimit,
	).Scan(&matchedCount, &dataIDs)
	if err != nil {
		// TODO - Send more specific errors?
		return pSearchResult, nex.NewError(nex.ResultCodes.DataStore.Unknown, err.Error())
	}

	pSearchResult.Result, errCode = GetSearchResultsByDataIDsWithResultOption(database, dataIDs, param.ResultOption)
	if errCode != nil {
		return pSearchResult, errCode
	}

	if !totalCountEnabled {
		pSearchResult.TotalCount = 0
		pSearchResult.TotalCountType = datastore_constants.SearchResultTotalDisabled
	} else if matchedCount > countedLimit {
		pSearchResult.TotalCount = types.UInt32(countedLimit)
		pSearchResult.TotalCountType = datastore_constants.SearchResultTotalMinimum
	} else {
		pSearchResult.TotalCount = types.UInt32(matchedCount)
		pSearchResult.TotalCountType = datastore_constants.SearchResultTotalExact
	}

	return pSearchResult, nil
}
