package datastore

import (
	"context"
	"slices"
	"strings"

	"github.com/PretendoNetwork/nex-go"
	"github.com/PretendoNetwork/nex-go/types"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
	"github.com/minio/minio-go/v7"
)

var commonProtocol *CommonProtocol

type CommonProtocol struct {
	server                                       nex.ServerInterface
	protocol                                     datastore.Interface
	S3Bucket                                     string
	s3DataKeyBase                                string
	s3NotifyKeyBase                              string
	RootCACert                                   []byte
	minIOClient                                  *minio.Client
	S3Presigner                                  S3PresignerInterface
	GetUserFriendPIDs                            func(pid uint32) []uint32
	GetObjectInfoByDataID                        func(dataID *types.PrimitiveU64) (*datastore_types.DataStoreMetaInfo, uint32)
	UpdateObjectPeriodByDataIDWithPassword       func(dataID *types.PrimitiveU64, dataType *types.PrimitiveU16, password *types.PrimitiveU64) uint32
	UpdateObjectMetaBinaryByDataIDWithPassword   func(dataID *types.PrimitiveU64, metaBinary *types.QBuffer, password *types.PrimitiveU64) uint32
	UpdateObjectDataTypeByDataIDWithPassword     func(dataID *types.PrimitiveU64, period *types.PrimitiveU16, password *types.PrimitiveU64) uint32
	GetObjectSizeByDataID                        func(dataID *types.PrimitiveU64) (uint32, uint32)
	UpdateObjectUploadCompletedByDataID          func(dataID *types.PrimitiveU64, uploadCompleted bool) uint32
	GetObjectInfoByPersistenceTargetWithPassword func(persistenceTarget *datastore_types.DataStorePersistenceTarget, password *types.PrimitiveU64) (*datastore_types.DataStoreMetaInfo, uint32)
	GetObjectInfoByDataIDWithPassword            func(dataID *types.PrimitiveU64, password *types.PrimitiveU64) (*datastore_types.DataStoreMetaInfo, uint32)
	S3GetRequestHeaders                          func() ([]*datastore_types.DataStoreKeyValue, uint32)
	S3PostRequestHeaders                         func() ([]*datastore_types.DataStoreKeyValue, uint32)
	InitializeObjectByPreparePostParam           func(ownerPID *types.PID, param *datastore_types.DataStorePreparePostParam) (uint64, uint32)
	InitializeObjectRatingWithSlot               func(dataID uint64, param *datastore_types.DataStoreRatingInitParamWithSlot) uint32
	RateObjectWithPassword                       func(dataID *types.PrimitiveU64, slot *types.PrimitiveU8, ratingValue *types.PrimitiveS32, accessPassword *types.PrimitiveU64) (*datastore_types.DataStoreRatingInfo, uint32)
	DeleteObjectByDataIDWithPassword             func(dataID *types.PrimitiveU64, password *types.PrimitiveU64) uint32
	DeleteObjectByDataID                         func(dataID *types.PrimitiveU64) uint32
	GetObjectInfosByDataStoreSearchParam         func(param *datastore_types.DataStoreSearchParam) ([]*datastore_types.DataStoreMetaInfo, uint32, uint32)
	GetObjectOwnerByDataID                       func(dataID *types.PrimitiveU64) (uint32, uint32)
}

func (c *CommonProtocol) S3StatObject(bucket, key string) (minio.ObjectInfo, error) {
	return c.minIOClient.StatObject(context.TODO(), bucket, key, minio.StatObjectOptions{})
}

func (c *CommonProtocol) S3ObjectSize(bucket, key string) (uint64, error) {
	info, err := c.S3StatObject(bucket, key)
	if err != nil {
		return 0, err
	}

	return uint64(info.Size), nil
}

func (c *CommonProtocol) VerifyObjectPermission(ownerPID, accessorPID *types.PID, permission *datastore_types.DataStorePermission) uint32 {
	if permission.Permission.Value > 3 {
		return nex.ResultCodes.DataStore.InvalidArgument
	}

	// * Owner can always access their own objects
	if ownerPID.Equals(accessorPID) {
		return 0
	}

	// * Allow anyone
	if permission.Permission.Value == 0 {
		return 0
	}

	// * Allow only friends of the owner
	if permission.Permission.Value == 1 {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friendsList := c.GetUserFriendPIDs(ownerPID.LegacyValue())

		if !slices.Contains(friendsList, accessorPID.LegacyValue()) {
			return nex.ResultCodes.DataStore.PermissionDenied
		}
	}

	// * Allow only users whose PIDs are defined in permission.RecipientIDs
	if permission.Permission.Value == 2 {
		if !permission.RecipientIDs.Contains(accessorPID) {
			return nex.ResultCodes.DataStore.PermissionDenied
		}
	}

	// * Allow only the owner
	if permission.Permission.Value == 3 {
		if !ownerPID.Equals(accessorPID) {
			return nex.ResultCodes.DataStore.PermissionDenied
		}
	}

	return 0
}

// SetDataKeyBase sets the base for the key to be used when uploading standard DataStore objects
func (c *CommonProtocol) SetDataKeyBase(base string) {
	// * Just in case someone passes a badly formatted key
	base = strings.TrimPrefix(base, "/")
	base = strings.TrimSuffix(base, "/")
	c.s3DataKeyBase = base
}

// SetNotifyKeyBase sets the base for the key to be used when uploading DataStore notification data
func (c *CommonProtocol) SetNotifyKeyBase(base string) {
	// * Just in case someone passes a badly formatted key
	base = strings.TrimPrefix(base, "/")
	base = strings.TrimSuffix(base, "/")
	c.s3NotifyKeyBase = base
}

// SetMinIOClient sets the MinIO S3 client
func (c *CommonProtocol) SetMinIOClient(client *minio.Client) {
	c.minIOClient = client
	c.S3Presigner = NewS3Presigner(c.minIOClient)
}

// NewCommonProtocol returns a new CommonProtocol
func NewCommonProtocol(protocol datastore.Interface) *CommonProtocol {
	protocol.SetHandlerDeleteObject(deleteObject)
	protocol.SetHandlerGetMeta(getMeta)
	protocol.SetHandlerGetMetas(getMetas)
	protocol.SetHandlerSearchObject(searchObject)
	protocol.SetHandlerRateObject(rateObject)
	protocol.SetHandlerPostMetaBinary(postMetaBinary)
	protocol.SetHandlerPreparePostObject(preparePostObject)
	protocol.SetHandlerPrepareGetObject(prepareGetObject)
	protocol.SetHandlerCompletePostObject(completePostObject)
	protocol.SetHandlerGetMetasMultipleParam(getMetasMultipleParam)
	protocol.SetHandlerCompletePostObjects(completePostObjects)
	protocol.SetHandlerChangeMeta(changeMeta)
	protocol.SetHandlerRateObjects(rateObjects)

	commonProtocol = &CommonProtocol{
		server:     protocol.Server(),
		protocol:   protocol,
		RootCACert: []byte{},
		S3GetRequestHeaders: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
			return []*datastore_types.DataStoreKeyValue{}, 0
		},
		S3PostRequestHeaders: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
			return []*datastore_types.DataStoreKeyValue{}, 0
		},
	}
	return commonProtocol
}
