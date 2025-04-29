package common_globals

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
	"github.com/PretendoNetwork/nex-go/v2/types"
	datastore_constants "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/constants"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/v2/datastore/types"
	"github.com/minio/minio-go/v7"
)

// S3GetObjectData stores the data required for a
// S3 client to request an object from the storage
// server
type S3GetObjectData struct {
	URL            *url.URL
	RequestHeaders map[string]string
	RootCACert     []byte
	Size           uint32
}

// S3GetObjectData stores the data required for a
// S3 client to upload an object to the storage
// server
type S3PostObjectData struct {
	URL            *url.URL
	FormData       map[string]string
	RequestHeaders map[string]string
	RootCACert     []byte
}

// S3Presigner defines the required methods for
// interacting with the S3 storage server through
// presigned requests
type S3Presigner interface {
	GetObject(bucket, key string, lifetime time.Duration) (*S3GetObjectData, error)
	PostObject(bucket, key string, lifetime time.Duration) (*S3PostObjectData, error)
}

// S3 represents an S3 configuration for a specific bucket
type S3 struct {
	Bucket    string
	KeyBase   string
	Presigner S3Presigner
}

// PresignGet creates a presigned GET request for a given object
func (s3 S3) PresignGet(key string, lifetime time.Duration) (*S3GetObjectData, error) {
	key = fmt.Sprintf("%s/%s", s3.KeyBase, key)
	return s3.Presigner.GetObject(s3.Bucket, key, lifetime)
}

// PresignGet creates a presigned POST request to upload a new object
func (s3 S3) PresignPost(key string, lifetime time.Duration) (*S3PostObjectData, error) {
	key = fmt.Sprintf("%s/%s", s3.KeyBase, key)
	return s3.Presigner.PostObject(s3.Bucket, key, lifetime)
}

// MinIOPresigner is an S3Presigner using MinIO as
// the S3 storage server
type MinIOPresigner struct {
	minioClient *minio.Client
}

// GetObject generates the MinIO presigned GET request data
func (mp MinIOPresigner) GetObject(bucket, key string, lifetime time.Duration) (*S3GetObjectData, error) {
	reqParams := make(url.Values)
	url, err := mp.minioClient.PresignedGetObject(context.Background(), bucket, key, lifetime, reqParams)
	if err != nil {
		return nil, err
	}

	return &S3GetObjectData{
		URL:            url,
		RequestHeaders: make(map[string]string), // TODO - Add a way to set these
		RootCACert:     make([]byte, 0),         // TODO - Add a way to set this
		Size:           0,                       // TODO - This is set in the NEX handler, but maybe it should be set here?
	}, nil
}

// PostObject generates the MinIO presigned POST request data
func (mp MinIOPresigner) PostObject(bucket, key string, lifetime time.Duration) (*S3PostObjectData, error) {
	policy := minio.NewPostPolicy()

	err := policy.SetBucket(bucket)
	if err != nil {
		return nil, err
	}

	err = policy.SetKey(key)
	if err != nil {
		return nil, err
	}

	err = policy.SetExpires(time.Now().UTC().Add(lifetime).UTC())
	if err != nil {
		return nil, err
	}

	url, formData, err := mp.minioClient.PresignedPostPolicy(context.Background(), policy)
	if err != nil {
		return nil, err
	}

	return &S3PostObjectData{
		URL:            url,
		FormData:       formData,
		RequestHeaders: make(map[string]string), // TODO - Add a way to set these
		RootCACert:     make([]byte, 0),         // TODO - Add a way to set this
	}, nil
}

// NewMinIOPresigner returns a new MinIOPresigner
func NewMinIOPresigner(minioClient *minio.Client) *MinIOPresigner {
	return &MinIOPresigner{
		minioClient: minioClient,
	}
}

// DataStoreManager manages a DataStore instance
type DataStoreManager struct {
	Database          *sql.DB
	Endpoint          *nex.PRUDPEndPoint
	S3                *S3
	GetUserFriendPIDs func(pid uint32) []uint32

	// * Some games may need to customize this behavior.
	// * Set as fields so they can be modified by the caller

	VerifyObjectAccessPermission func(requesterPID types.PID, metaInfo datastore_types.DataStoreMetaInfo, objectAccessPassword, requesterAccessPassword types.UInt64) *nex.Error
	VerifyObjectUpdatePermission func(requesterPID types.PID, metaInfo datastore_types.DataStoreMetaInfo, objectUpdatePassword, requesterUpdatePassword types.UInt64) *nex.Error
	VerifyObjectPermission       func(ownerPID, requesterPID types.PID, permission datastore_types.DataStorePermission, objectPassword, requesterPassword types.UInt64) *nex.Error
	ValidateExtraData            func(extraData types.List[types.String]) *nex.Error
}

// SetS3Config sets the S3 config for the DataStoreManager.
//
// Only one bucket can be configured at a time
func (dsm *DataStoreManager) SetS3Config(bucket, keyBase string, presigner S3Presigner) {
	dsm.S3 = &S3{
		Bucket:    bucket,
		KeyBase:   keyBase,
		Presigner: presigner,
	}
}

// verifyObjectAccessPermission is the default implementation that verifies that a request can access a given object
func (dsm DataStoreManager) verifyObjectAccessPermission(requesterPID types.PID, metaInfo datastore_types.DataStoreMetaInfo, objectAccessPassword, requesterAccessPassword types.UInt64) *nex.Error {
	return dsm.VerifyObjectPermission(metaInfo.OwnerID, requesterPID, metaInfo.Permission, objectAccessPassword, requesterAccessPassword)
}

// verifyObjectUpdatePermission is the default implementation that verifies that a request can update a given object
func (dsm DataStoreManager) verifyObjectUpdatePermission(requesterPID types.PID, metaInfo datastore_types.DataStoreMetaInfo, objectUpdatePassword, requesterUpdatePassword types.UInt64) *nex.Error {
	return dsm.VerifyObjectPermission(metaInfo.OwnerID, requesterPID, metaInfo.DelPermission, objectUpdatePassword, requesterUpdatePassword)
}

// verifyObjectPermission is the default implementation that verifies that a given set of permissions is allowed
func (dsm DataStoreManager) verifyObjectPermission(ownerPID, requesterPID types.PID, permission datastore_types.DataStorePermission, objectPassword, requesterPassword types.UInt64) *nex.Error {
	if permission.Permission > types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "change_error")
	}

	// * If a password is provided, and is correct, then bypass
	// * permissions checks
	if uint64(requesterPassword) != datastore_constants.InvalidPassword && requesterPassword == objectPassword {
		return nil
	}

	// * Owner can always interact with their own objects
	if ownerPID.Equals(requesterPID) {
		return nil
	}

	// * Standard permission checks
	var err *nex.Error

	if permission.Permission == types.UInt8(datastore_constants.PermissionPublic) {
		return nil
	}

	if permission.Permission == types.UInt8(datastore_constants.PermissionFriend) {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friendsList := dsm.GetUserFriendPIDs(uint32(ownerPID))

		if !slices.Contains(friendsList, uint32(requesterPID)) {
			err = nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}
	}

	if permission.Permission == types.UInt8(datastore_constants.PermissionSpecified) {
		if !permission.RecipientIDs.Contains(requesterPID) {
			err = nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}
	}

	if permission.Permission == types.UInt8(datastore_constants.PermissionPrivate) {
		if !ownerPID.Equals(requesterPID) {
			err = nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}
	}

	if permission.Permission == types.UInt8(datastore_constants.PermissionSpecifiedFriend) {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friendsList := dsm.GetUserFriendPIDs(uint32(ownerPID))

		if !slices.Contains(friendsList, uint32(requesterPID)) {
			err = nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}

		if !permission.RecipientIDs.Contains(requesterPID) {
			err = nex.NewError(nex.ResultCodes.DataStore.PermissionDenied, "change_error")
		}
	}

	if err != nil && uint64(requesterPassword) != datastore_constants.InvalidPassword {
		err = nex.NewError(nex.ResultCodes.DataStore.InvalidPassword, "change_error")
	}

	return err
}

// validateExtraData is the default implementation validates the `extraData` list seen in
// `DataStorePreparePostParam`, `DataStorePrepareUpdateParam`,
// and `DataStorePrepareGetParam`.
//
// NOTE: UNOFFICIAL BEHAVIOR! THIS USES HEURISTICS BASED ON HOW SEVERAL
// GAMES CREATE THIS DATA. THE OFFICIAL SERVERS DID NOT VALIDATE THIS
// DATA AT ALL, WE DO SO FOR SANITY AND SAFETY!
func (dsm DataStoreManager) validateExtraData(extraData types.List[types.String]) *nex.Error {
	// * These checks are based on observed behaviour in
	// * Animal Crossing: New Leaf (3DS) and Xenoblade (Wii U).
	// *
	// * `extraData` seems to contain data about the device
	// * type and region? Unsure what the real purpose of
	// * this data is. The structure of `extraData` seems
	// * consistent across multiple games, however.
	// *
	// * Some notes on the structure as seen in
	// * `nn::nex::DataStoreLogicServerClient::CreateExtraData`
	// * from Xenoblade on the Wii U (3DS handles parts slightly different):
	// *
	// * - The 1st element is the platform ("CTR" for the 3DS (all models), "WUP" for the Wii U)
	// * - The 2nd element is the "platform region" number. CFG_Region on the 3DS, MCPRegion on Wii U.
	// *   Gotten from `SCIGetPlatformRegion` on Wii U, likely `CFGU_SecureInfoGetRegion` on 3DS?
	// * - The 3rd element is the "platform region" name. This takes the platform region number
	// *   from the 2nd element and finds it's corresponding string name. For example, "3" on 3DS
	// *   would use "AUS" here, and 64 on Wii U would use "TWN" here. If the number is not valid,
	// *   the string "Invalid" is used
	// * - The 4th element is the "platform country" number. It comes from `SCIGetCafeCountry` on
	// *   Wii U and seems to line up with the standard region IDs
	// *   (https://nintendo-wiki.pretendo.network/docs/misc/region-ids)
	// * - The 5th element is the "platform country" code. It comes from `SCIGetCountryCodeA2(cafe_country)`
	// *   on Wii U and is just the ISO A2 country code. For example country code 110 (UK) would
	// *   map to "GB". Seems to line up with the `X-Nintendo-Country` NNAS header
	// * - The 6th element is always empty. AC:NL dumps have nothing here, and Xenoblade has no
	// *   code to populate it. Maybe it's reserved for something? Idk
	// * - Every string has a max length length of 8 characters ("Invalid" is 7 characters, plus the null byte)
	// *
	// * Example from Animal Crossing: New Leaf:
	// *
	// * extraData (List<String> length 6)
	// * 	extraData[0] (String): CTR
	// * 	extraData[1] (String): 2
	// * 	extraData[2] (String): EUR
	// * 	extraData[3] (String): 110
	// * 	extraData[4] (String): GB
	// * 	extraData[5] (String):

	// * `extraData` was added in NEX 3.5. If empty, assume
	// * legacy client and ignore it
	if len(extraData) == 0 {
		return nil
	}

	// * Always 6 elements in length
	if len(extraData) != 6 {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, not 6 elements")
	}

	// * The max size of a string here is always 7.
	// * The game allocates 8 bytes, but the last
	// * is reserved for the trailing null byte
	for _, str := range extraData {
		if len(str) > 7 {
			return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, string too long")
		}
	}

	// * The 3DS and Wii U use different platform region IDs
	platformRegionIDs := map[string]map[string]int{
		"CTR": { // * CFG_Region https://github.com/devkitPro/libctru/blob/4dcc4cc65468fa6a28733ddeabdd650c6dda7f41/libctru/include/3ds/services/cfgu.h#L9
			"JPN": 0,
			"USA": 1,
			"EUR": 2,
			"AUS": 3,
			"CHN": 4,
			"KOR": 5,
			"TWN": 6,
		},
		"WUP": { // * MCPRegion https://github.com/devkitPro/wut/blob/b35e595a6e4b69225a8e014046944d9157248513/include/coreinit/mcp.h#L87
			"Invalid": 0, // * Not seen in MCPRegion, but seen in Xenoblade on Wii U
			"JPN":     1,
			"USA":     2,
			"EUR":     4,
			"AUS":     8, // * Not seen in MCPRegion, but seen in Xenoblade on Wii U
			"CHN":     16,
			"KOR":     32,
			"TWN":     64,
		},
	}

	platform := string(extraData[0])
	platformRegionIDStr := string(extraData[1])
	platformRegionName := string(extraData[2])

	// TODO - Based on decomp of Xenoblade, it looks like some of these fields CAN be empty
	//        if something goes wrong. Empty fields are not currently supported

	if _, ok := platformRegionIDs[platform]; !ok {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, invalid platform type")
	}

	platformRegionID, err := strconv.Atoi(platformRegionIDStr)
	if err != nil {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, platform region ID not a number")
	}

	if _, ok := platformRegionIDs[platform][platformRegionName]; !ok {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, invalid platform region name")
	}

	if platformRegionIDs[platform][platformRegionName] != platformRegionID {
		return nex.NewError(nex.ResultCodes.DataStore.InvalidArgument, "invalid extraData, platform ID in data does not match region name")
	}

	// * extraData[3] is the numeric "region ID" (top-level). There are thousands of these.
	// * For example, the string "110" represents the UK (country code GB).
	// * See https://nintendo-wiki.pretendo.network/docs/misc/region-ids
	// * for a full list
	// TODO - Verify extraData[3]. Too many to check right now

	// * extraData[4] is the A2 country code for the region in extraData[3].
	// * For example, if extraData[3] is "110" then extraData[4] is "GB"
	// TODO - Verify extraData[4]. Cannot do so until extraData[3] is also checked

	// * extraData[5] is always empty? Might be reserved, no checks for now

	return nil
}

// NewDataStoreManager returns a new DataStoreManager
func NewDataStoreManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *DataStoreManager {
	dsm := &DataStoreManager{
		Database: db,
		Endpoint: endpoint,
	}

	dsm.VerifyObjectAccessPermission = dsm.verifyObjectAccessPermission
	dsm.VerifyObjectUpdatePermission = dsm.verifyObjectUpdatePermission
	dsm.VerifyObjectPermission = dsm.verifyObjectPermission
	dsm.ValidateExtraData = dsm.validateExtraData

	return dsm
}
