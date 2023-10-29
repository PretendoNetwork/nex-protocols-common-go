package datastore

import (
	"net/url"
	"strings"
	"time"

	nex "github.com/PretendoNetwork/nex-go"
	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/globals"
	datastore "github.com/PretendoNetwork/nex-protocols-go/datastore"
	datastore_super_mario_maker "github.com/PretendoNetwork/nex-protocols-go/datastore/super-mario-maker"
	datastore_types "github.com/PretendoNetwork/nex-protocols-go/datastore/types"
)

var commonDataStoreProtocol *CommonDataStoreProtocol

type CommonDataStoreProtocol struct {
	server                  *nex.Server
	DefaultProtocol         *datastore.Protocol
	SuperMarioMakerProtocol *datastore_super_mario_maker.Protocol

	s3Bucket                                            string
	rootCACert                                          []byte
	getObjectInfoByDataIDHandler                        func(dataID uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	verifyObjectPermissionHandler                       func(ownerPID uint32, accessorPID uint32, permission *datastore_types.DataStorePermission) uint32
	updateObjectPeriodByDataIDWithPasswordHandler       func(dataID uint64, dataType uint16, password uint64) uint32
	updateObjectMetaBinaryByDataIDWithPasswordHandler   func(dataID uint64, metaBinary []byte, password uint64) uint32
	updateObjectDataTypeByDataIDWithPasswordHandler     func(dataID uint64, period uint16, password uint64) uint32
	s3ObjectSizeHandler                                 func(bucket string, key string) (uint64, error)
	getObjectSizeDataIDHandler                          func(dataID uint64) (uint32, uint32)
	updateObjectUploadCompletedByDataIDHandler          func(dataID uint64, uploadCompleted bool) uint32
	getObjectInfoByPersistenceTargetWithPasswordHandler func(persistenceTarget *datastore_types.DataStorePersistenceTarget, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	getObjectInfoByDataIDWithPasswordHandler            func(dataID uint64, password uint64) (*datastore_types.DataStoreMetaInfo, uint32)
	presignGetObjectHandler                             func(bucket string, key string, lifetime time.Duration) (*url.URL, error)
	presignPostObjectHandler                            func(bucket string, key string, lifetime time.Duration) (*url.URL, map[string]string, error)
	s3GetRequestHeadersHandler                          func() ([]*datastore_types.DataStoreKeyValue, uint32)
	s3PostRequestHeadersHandler                         func() ([]*datastore_types.DataStoreKeyValue, uint32)
	initializeObjectByPreparePostParamHandler           func(ownerPID uint32, param *datastore_types.DataStorePreparePostParam) (uint64, uint32)
	initializeObjectRatingWithSlotHandler               func(dataID uint64, param *datastore_types.DataStoreRatingInitParamWithSlot) uint32
	rateObjectWithPasswordHandler                       func(dataID uint64, slot uint8, ratingValue int32, accessPassword uint64) (*datastore_types.DataStoreRatingInfo, uint32)
	deleteObjectByDataIDWithPasswordHandler             func(dataID uint64, password uint64) uint32
	getObjectInfosByDataStoreSearchParamHandler         func(param *datastore_types.DataStoreSearchParam) ([]*datastore_types.DataStoreMetaInfo, uint32, uint32)
}

// SetS3Bucket sets the S3 bucket
func (c *CommonDataStoreProtocol) SetS3Bucket(bucket string) {
	c.s3Bucket = bucket
}

// SetRootCACert sets the S3 root CA
func (c *CommonDataStoreProtocol) SetRootCACert(rootCACert []byte) {
	c.rootCACert = rootCACert
}

// GetObjectInfoByDataID sets the GetObjectInfoByDataID handler function
func (c *CommonDataStoreProtocol) GetObjectInfoByDataID(handler func(dataID uint64) (*datastore_types.DataStoreMetaInfo, uint32)) {
	c.getObjectInfoByDataIDHandler = handler
}

// VerifyObjectPermission sets the VerifyObjectPermission handler function
func (c *CommonDataStoreProtocol) VerifyObjectPermission(handler func(ownerPID uint32, accessorPID uint32, permission *datastore_types.DataStorePermission) uint32) {
	c.verifyObjectPermissionHandler = handler
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

// S3ObjectSize sets the S3ObjectSize handler function
func (c *CommonDataStoreProtocol) S3ObjectSize(handler func(bucket string, key string) (uint64, error)) {
	c.s3ObjectSizeHandler = handler
}

// GetObjectSizeDataID sets the GetObjectSizeDataID handler function
func (c *CommonDataStoreProtocol) GetObjectSizeDataID(handler func(dataID uint64) (uint32, uint32)) {
	c.getObjectSizeDataIDHandler = handler
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

// PresignGetObject sets the PresignGetObject handler function
func (c *CommonDataStoreProtocol) PresignGetObject(handler func(bucket string, key string, lifetime time.Duration) (*url.URL, error)) {
	c.presignGetObjectHandler = handler
}

// PresignPostObject sets the PresignPostObject handler function
func (c *CommonDataStoreProtocol) PresignPostObject(handler func(bucket string, key string, lifetime time.Duration) (*url.URL, map[string]string, error)) {
	c.presignPostObjectHandler = handler
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
