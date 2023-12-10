package datastore

import (
	"context"
	"slices"
	"strings"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_super_mario_maker "github.com/PretendoNetwork/nex-protocols-go/datastore/super-mario-maker"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
	"github.com/minio/minio-go/v7"
)

var commonDataStoreProtocol *CommonDataStoreProtocol

type CommonDataStoreProtocol struct {
	server                  nex.ServerInterface
	DefaultProtocol         *datastore.Protocol
	SuperMarioMakerProtocol *datastore_super_mario_maker.Protocol

	S3Bucket                                     string
	s3DataKeyBase                                string
	s3NotifyKeyBase                              string
	RootCACert                                   []byte
	minIOClient                                  *minio.Client
	S3Presigner                                  S3PresignerInterface
	GetUserFriendPIDs                            func(pid uint32) []uint32
	GetObjectInfoByDataID                        func(dataID uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	UpdateObjectPeriodByDataIDWithPassword       func(dataID uint64, dataType uint16, password uint64) uint32
	UpdateObjectMetaBinaryByDataIDWithPassword   func(dataID uint64, metaBinary []byte, password uint64) uint32
	UpdateObjectDataTypeByDataIDWithPassword     func(dataID uint64, period uint16, password uint64) uint32
	GetObjectSizeByDataID                        func(dataID uint64) (uint32, uint32)
	UpdateObjectUploadCompletedByDataID          func(dataID uint64, uploadCompleted bool) uint32
	GetObjectInfoByPersistenceTargetWithPassword func(persistenceTarget *datastore_types.DataStorePersistenceTarget, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	GetObjectInfoByDataIDWithPassword            func(dataID uint64, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	S3GetRequestHeaders                          func() ([]*datastore_types.DataStoreKeyValue, uint32)
	S3PostRequestHeaders                         func() ([]*datastore_types.DataStoreKeyValue, uint32)
	InitializeObjectByPreparePostParam           func(ownerPID uint32, param *datastore_types.DataStorePreparePostParam) (uint64, uint32)
	InitializeObjectRatingWithSlot               func(dataID uint64, param *datastore_types.DataStoreRatingInitParamWithSlot) uint32
	RateObjectWithPassword                       func(dataID uint64, slot uint8, ratingValue int32, accessPassword uint64) (*datastore_types.DataStoreRatingInfo, uint32)
	DeleteObjectByDataIDWithPassword             func(dataID uint64, password uint64) uint32
	DeleteObjectByDataID                         func(dataID uint64) uint32
	GetObjectInfosByDataStoreSearchParam         func(param *datastore_types.DataStoreSearchParam) ([]*datastore_types.DataStoreMetaInfo, uint32, uint32)
	GetObjectOwnerByDataID                       func(dataID uint64) (uint32, uint32)
}

func (c *CommonDataStoreProtocol) S3StatObject(bucket, key string) (minio.ObjectInfo, error) {
	return c.minIOClient.StatObject(context.TODO(), bucket, key, minio.StatObjectOptions{})
}

func (c *CommonDataStoreProtocol) S3ObjectSize(bucket, key string) (uint64, error) {
	info, err := c.S3StatObject(bucket, key)
	if err != nil {
		return 0, err
	}

	return uint64(info.Size), nil
}

func (c *CommonDataStoreProtocol) VerifyObjectPermission(ownerPID, accessorPID *nex.PID, permission *datastore_types.DataStorePermission) uint32 {
	if permission.Permission > 3 {
		return nex.Errors.DataStore.InvalidArgument
	}

	// * Owner can always access their own objects
	if ownerPID.Equals(accessorPID) {
		return 0
	}

	// * Allow anyone
	if permission.Permission == 0 {
		return 0
	}

	// * Allow only friends of the owner
	if permission.Permission == 1 {
		// TODO - This assumes a legacy client. Will not work on the Switch
		friendsList := c.GetUserFriendPIDs(ownerPID.LegacyValue())

		if !slices.Contains(friendsList, accessorPID.LegacyValue()) {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	// * Allow only users whose PIDs are defined in permission.RecipientIDs
	if permission.Permission == 2 {
		if !common_globals.ContainsPID(permission.RecipientIDs, accessorPID) {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	// * Allow only the owner
	if permission.Permission == 3 {
		if !ownerPID.Equals(accessorPID) {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	return 0
}

// SetDataKeyBase sets the base for the key to be used when uploading standard DataStore objects
func (c *CommonDataStoreProtocol) SetDataKeyBase(base string) {
	// * Just in case someone passes a badly formatted key
	base = strings.TrimPrefix(base, "/")
	base = strings.TrimSuffix(base, "/")
	c.s3DataKeyBase = base
}

// SetNotifyKeyBase sets the base for the key to be used when uploading DataStore notification data
func (c *CommonDataStoreProtocol) SetNotifyKeyBase(base string) {
	// * Just in case someone passes a badly formatted key
	base = strings.TrimPrefix(base, "/")
	base = strings.TrimSuffix(base, "/")
	c.s3NotifyKeyBase = base
}

// SetMinIOClient sets the MinIO S3 client
func (c *CommonDataStoreProtocol) SetMinIOClient(client *minio.Client) {
	c.minIOClient = client
	c.S3Presigner = NewS3Presigner(c.minIOClient)
}

func initDefault(c *CommonDataStoreProtocol) {
	c.DefaultProtocol = datastore.NewProtocol(c.server)
	c.DefaultProtocol.DeleteObject = deleteObject
	c.DefaultProtocol.GetMeta = getMeta
	c.DefaultProtocol.GetMetas = getMetas
	c.DefaultProtocol.SearchObject = searchObject
	c.DefaultProtocol.RateObject = rateObject
	c.DefaultProtocol.PostMetaBinary = postMetaBinary
	c.DefaultProtocol.PreparePostObject = preparePostObject
	c.DefaultProtocol.PrepareGetObject = prepareGetObject
	c.DefaultProtocol.CompletePostObject = completePostObject
	c.DefaultProtocol.GetMetasMultipleParam = getMetasMultipleParam
	c.DefaultProtocol.CompletePostObjects = completePostObjects
	c.DefaultProtocol.ChangeMeta = changeMeta
	c.DefaultProtocol.RateObjects = rateObjects
}

func initSuperMarioMaker(c *CommonDataStoreProtocol) {
	c.SuperMarioMakerProtocol = datastore_super_mario_maker.NewProtocol(c.server)
	c.SuperMarioMakerProtocol.DeleteObject = deleteObject
	c.SuperMarioMakerProtocol.GetMeta = getMeta
	c.SuperMarioMakerProtocol.GetMetas = getMetas
	c.SuperMarioMakerProtocol.SearchObject = searchObject
	c.SuperMarioMakerProtocol.RateObject = rateObject
	c.SuperMarioMakerProtocol.PostMetaBinary = postMetaBinary
	c.SuperMarioMakerProtocol.PreparePostObject = preparePostObject
	c.SuperMarioMakerProtocol.PrepareGetObject = prepareGetObject
	c.SuperMarioMakerProtocol.CompletePostObject = completePostObject
	c.SuperMarioMakerProtocol.GetMetasMultipleParam = getMetasMultipleParam
	c.SuperMarioMakerProtocol.CompletePostObjects = completePostObjects
	c.SuperMarioMakerProtocol.ChangeMeta = changeMeta
	c.SuperMarioMakerProtocol.RateObjects = rateObjects
}

// NewCommonDataStoreProtocol returns a new CommonDataStoreProtocol
func NewCommonDataStoreProtocol(server nex.ServerInterface) *CommonDataStoreProtocol {
	commonDataStoreProtocol = &CommonDataStoreProtocol{
		server:     server,
		RootCACert: []byte{},
		S3GetRequestHeaders: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
			return []*datastore_types.DataStoreKeyValue{}, 0
		},
		S3PostRequestHeaders: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
			return []*datastore_types.DataStoreKeyValue{}, 0
		},
	}

	patch := server.DataStoreProtocolVersion().GameSpecificPatch

	if strings.EqualFold(patch, "AMAJ") {
		common_globals.Logger.Info("Using Super Mario Maker DataStore protocol")
		initSuperMarioMaker(commonDataStoreProtocol)
	} else {
		if patch != "" {
			common_globals.Logger.Infof("DataStore version patch %q not recognized", patch)
		}

		common_globals.Logger.Info("Using default DataStore protocol")
		initDefault(commonDataStoreProtocol)
	}

	return commonDataStoreProtocol
}
