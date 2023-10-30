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
	server                  *nex.Server
	DefaultProtocol         *datastore.Protocol
	SuperMarioMakerProtocol *datastore_super_mario_maker.Protocol

	s3Bucket                                            string
	s3DataKeyBase                                       string
	s3NotifyKeyBase                                     string
	rootCACert                                          []byte
	MinIOClient                                         *minio.Client
	S3Presigner                                         S3PresignerInterface
	getUserFriendPIDsHandler                            func(pid uint32) []uint32
	getObjectInfoByDataIDHandler                        func(dataID uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	updateObjectPeriodByDataIDWithPasswordHandler       func(dataID uint64, dataType uint16, password uint64) uint32
	updateObjectMetaBinaryByDataIDWithPasswordHandler   func(dataID uint64, metaBinary []byte, password uint64) uint32
	updateObjectDataTypeByDataIDWithPasswordHandler     func(dataID uint64, period uint16, password uint64) uint32
	getObjectSizeByDataIDHandler                        func(dataID uint64) (uint32, uint32)
	updateObjectUploadCompletedByDataIDHandler          func(dataID uint64, uploadCompleted bool) uint32
	getObjectInfoByPersistenceTargetWithPasswordHandler func(persistenceTarget *datastore_types.DataStorePersistenceTarget, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	getObjectInfoByDataIDWithPasswordHandler            func(dataID uint64, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	s3GetRequestHeadersHandler                          func() ([]*datastore_types.DataStoreKeyValue, uint32)
	s3PostRequestHeadersHandler                         func() ([]*datastore_types.DataStoreKeyValue, uint32)
	initializeObjectByPreparePostParamHandler           func(ownerPID uint32, param *datastore_types.DataStorePreparePostParam) (uint64, uint32)
	initializeObjectRatingWithSlotHandler               func(dataID uint64, param *datastore_types.DataStoreRatingInitParamWithSlot) uint32
	rateObjectWithPasswordHandler                       func(dataID uint64, slot uint8, ratingValue int32, accessPassword uint64) (*datastore_types.DataStoreRatingInfo, uint32)
	deleteObjectByDataIDWithPasswordHandler             func(dataID uint64, password uint64) uint32
	deleteObjectByDataIDHandler                         func(dataID uint64) uint32
	getObjectInfosByDataStoreSearchParamHandler         func(param *datastore_types.DataStoreSearchParam) ([]*datastore_types.DataStoreMetaInfo, uint32, uint32)
}

func (c *CommonDataStoreProtocol) S3StatObject(bucket, key string) (minio.ObjectInfo, error) {
	return c.MinIOClient.StatObject(context.TODO(), bucket, key, minio.StatObjectOptions{})
}

func (c *CommonDataStoreProtocol) S3ObjectSize(bucket, key string) (uint64, error) {
	info, err := c.S3StatObject(bucket, key)
	if err != nil {
		return 0, err
	}

	return uint64(info.Size), nil
}

func (c *CommonDataStoreProtocol) VerifyObjectPermission(ownerPID, accessorPID uint32, permission *datastore_types.DataStorePermission) uint32 {
	if permission.Permission > 3 {
		return nex.Errors.DataStore.InvalidArgument
	}

	// * Owner can always access their own objects
	if ownerPID == accessorPID {
		return 0
	}

	// * Allow anyone
	if permission.Permission == 0 {
		return 0
	}

	// * Allow only friends of the owner
	if permission.Permission == 1 {
		friendsList := c.getUserFriendPIDsHandler(ownerPID)

		if !slices.Contains(friendsList, accessorPID) {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	// * Allow only users whose PIDs are defined in permission.RecipientIDs
	if permission.Permission == 2 {
		if !slices.Contains(permission.RecipientIDs, accessorPID) {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	// * Allow only the owner
	if permission.Permission == 3 {
		if ownerPID != accessorPID {
			return nex.Errors.DataStore.PermissionDenied
		}
	}

	return 0
}

// SetS3Bucket sets the S3 bucket
func (c *CommonDataStoreProtocol) SetS3Bucket(bucket string) {
	c.s3Bucket = bucket
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

// SetRootCACert sets the S3 root CA
func (c *CommonDataStoreProtocol) SetRootCACert(rootCACert []byte) {
	c.rootCACert = rootCACert
}

// SetMinIOClient sets the MinIO S3 client
func (c *CommonDataStoreProtocol) SetMinIOClient(client *minio.Client) {
	c.MinIOClient = client
	c.SetS3Presigner(NewS3Presigner(c.MinIOClient))
}

// SetS3Presigner sets the struct which creates presigned S3 URLs
func (c *CommonDataStoreProtocol) SetS3Presigner(presigner S3PresignerInterface) {
	c.S3Presigner = presigner
}

// SetGetUserFriendPIDs sets the handler for a function which gets a list of a users friend PIDs
func (c *CommonDataStoreProtocol) SetGetUserFriendPIDs(handler func(pid uint32) []uint32) {
	c.getUserFriendPIDsHandler = handler
}

// GetObjectInfoByDataID sets the GetObjectInfoByDataID handler function
func (c *CommonDataStoreProtocol) GetObjectInfoByDataID(handler func(dataID uint64) (*datastore_types.DataStoreMetaInfo, uint32)) {
	c.getObjectInfoByDataIDHandler = handler
}

// UpdateObjectPeriodByDataIDWithPassword sets the UpdateObjectPeriodByDataIDWithPassword handler function
func (c *CommonDataStoreProtocol) UpdateObjectPeriodByDataIDWithPassword(handler func(dataID uint64, dataType uint16, password uint64) uint32) {
	c.updateObjectPeriodByDataIDWithPasswordHandler = handler
}

// UpdateObjectMetaBinaryByDataIDWithPassword sets the UpdateObjectMetaBinaryByDataIDWithPassword handler function
func (c *CommonDataStoreProtocol) UpdateObjectMetaBinaryByDataIDWithPassword(handler func(dataID uint64, metaBinary []byte, password uint64) uint32) {
	c.updateObjectMetaBinaryByDataIDWithPasswordHandler = handler
}

// UpdateObjectDataTypeByDataIDWithPassword sets the UpdateObjectDataTypeByDataIDWithPassword handler function
func (c *CommonDataStoreProtocol) UpdateObjectDataTypeByDataIDWithPassword(handler func(dataID uint64, period uint16, password uint64) uint32) {
	c.updateObjectDataTypeByDataIDWithPasswordHandler = handler
}

// GetObjectSizeDataID sets the GetObjectSizeDataID handler function
func (c *CommonDataStoreProtocol) GetObjectSizeDataID(handler func(dataID uint64) (uint32, uint32)) {
	c.getObjectSizeByDataIDHandler = handler
}

// UpdateObjectUploadCompletedByDataID sets the UpdateObjectUploadCompletedByDataID handler function
func (c *CommonDataStoreProtocol) UpdateObjectUploadCompletedByDataID(handler func(dataID uint64, uploadCompleted bool) uint32) {
	c.updateObjectUploadCompletedByDataIDHandler = handler
}

// GetObjectInfoByPersistenceTargetWithPassword sets the GetObjectInfoByPersistenceTargetWithPassword handler function
func (c *CommonDataStoreProtocol) GetObjectInfoByPersistenceTargetWithPassword(handler func(persistenceTarget *datastore_types.DataStorePersistenceTarget, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)) {
	c.getObjectInfoByPersistenceTargetWithPasswordHandler = handler
}

// GetObjectInfoByDataIDWithPassword sets the GetObjectInfoByDataIDWithPassword handler function
func (c *CommonDataStoreProtocol) GetObjectInfoByDataIDWithPassword(handler func(dataID uint64, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)) {
	c.getObjectInfoByDataIDWithPasswordHandler = handler
}

// S3GetRequestHeaders sets the S3GetRequestHeaders handler function
func (c *CommonDataStoreProtocol) S3GetRequestHeaders(handler func() ([]*datastore_types.DataStoreKeyValue, uint32)) {
	c.s3GetRequestHeadersHandler = handler
}

// S3PostRequestHeaders sets the S3PostRequestHeaders handler function
func (c *CommonDataStoreProtocol) S3PostRequestHeaders(handler func() ([]*datastore_types.DataStoreKeyValue, uint32)) {
	c.s3PostRequestHeadersHandler = handler
}

// InitializeObjectByPreparePostParam sets the InitializeObjectByPreparePostParam handler function
func (c *CommonDataStoreProtocol) InitializeObjectByPreparePostParam(handler func(ownerPID uint32, param *datastore_types.DataStorePreparePostParam) (uint64, uint32)) {
	c.initializeObjectByPreparePostParamHandler = handler
}

// InitializeObjectRatingWithSlot sets the InitializeObjectRatingWithSlot handler function
func (c *CommonDataStoreProtocol) InitializeObjectRatingWithSlot(handler func(dataID uint64, param *datastore_types.DataStoreRatingInitParamWithSlot) uint32) {
	c.initializeObjectRatingWithSlotHandler = handler
}

// RateObjectWithPassword sets the RateObjectWithPassword handler function
func (c *CommonDataStoreProtocol) RateObjectWithPassword(handler func(dataID uint64, slot uint8, ratingValue int32, accessPassword uint64) (*datastore_types.DataStoreRatingInfo, uint32)) {
	c.rateObjectWithPasswordHandler = handler
}

// DeleteObjectByDataIDWithPassword sets the DeleteObjectByDataIDWithPassword handler function
func (c *CommonDataStoreProtocol) DeleteObjectByDataIDWithPassword(handler func(dataID uint64, password uint64) uint32) {
	c.deleteObjectByDataIDWithPasswordHandler = handler
}

// DeleteObjectByDataID sets the DeleteObjectByDataID handler function
func (c *CommonDataStoreProtocol) DeleteObjectByDataID(handler func(dataID uint64) uint32) {
	c.deleteObjectByDataIDHandler = handler
}

// GetObjectInfosByDataStoreSearchParam sets the GetObjectInfosByDataStoreSearchParam handler function
func (c *CommonDataStoreProtocol) GetObjectInfosByDataStoreSearchParam(handler func(param *datastore_types.DataStoreSearchParam) ([]*datastore_types.DataStoreMetaInfo, uint32, uint32)) {
	c.getObjectInfosByDataStoreSearchParamHandler = handler
}

func initDefault(c *CommonDataStoreProtocol) {
	c.DefaultProtocol = datastore.NewProtocol(c.server)
	c.DefaultProtocol.DeleteObject(deleteObject)
	c.DefaultProtocol.GetMeta(getMeta)
	c.DefaultProtocol.GetMetas(getMetas)
	c.DefaultProtocol.SearchObject(searchObject)
	c.DefaultProtocol.RateObject(rateObject)
	c.DefaultProtocol.PostMetaBinary(postMetaBinary)
	c.DefaultProtocol.PreparePostObject(preparePostObject)
	c.DefaultProtocol.PrepareGetObject(prepareGetObject)
	c.DefaultProtocol.CompletePostObject(completePostObject)
	c.DefaultProtocol.GetMetasMultipleParam(getMetasMultipleParam)
	c.DefaultProtocol.CompletePostObjects(completePostObjects)
	c.DefaultProtocol.ChangeMeta(changeMeta)
	c.DefaultProtocol.RateObjects(rateObjects)
}

func initSuperMarioMaker(c *CommonDataStoreProtocol) {
	c.SuperMarioMakerProtocol = datastore_super_mario_maker.NewProtocol(c.server)
	c.SuperMarioMakerProtocol.DeleteObject(deleteObject)
	c.SuperMarioMakerProtocol.GetMeta(getMeta)
	c.SuperMarioMakerProtocol.GetMetas(getMetas)
	c.SuperMarioMakerProtocol.SearchObject(searchObject)
	c.SuperMarioMakerProtocol.RateObject(rateObject)
	c.SuperMarioMakerProtocol.PostMetaBinary(postMetaBinary)
	c.SuperMarioMakerProtocol.PreparePostObject(preparePostObject)
	c.SuperMarioMakerProtocol.PrepareGetObject(prepareGetObject)
	c.SuperMarioMakerProtocol.CompletePostObject(completePostObject)
	c.SuperMarioMakerProtocol.GetMetasMultipleParam(getMetasMultipleParam)
	c.SuperMarioMakerProtocol.CompletePostObjects(completePostObjects)
	c.SuperMarioMakerProtocol.ChangeMeta(changeMeta)
	c.SuperMarioMakerProtocol.RateObjects(rateObjects)
}

// NewCommonDataStoreProtocol returns a new CommonDataStoreProtocol
func NewCommonDataStoreProtocol(server *nex.Server) *CommonDataStoreProtocol {
	commonDataStoreProtocol = &CommonDataStoreProtocol{
		server:     server,
		rootCACert: []byte{},
		s3GetRequestHeadersHandler: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
			return []*datastore_types.DataStoreKeyValue{}, 0
		},
		s3PostRequestHeadersHandler: func() ([]*datastore_types.DataStoreKeyValue, uint32) {
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
