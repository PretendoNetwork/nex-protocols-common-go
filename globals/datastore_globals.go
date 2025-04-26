package common_globals

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/PretendoNetwork/nex-go/v2"
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
	Database *sql.DB
	Endpoint *nex.PRUDPEndPoint
	S3       *S3
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

// NewDataStoreManager returns a new DataStoreManager
func NewDataStoreManager(endpoint *nex.PRUDPEndPoint, db *sql.DB) *DataStoreManager {
	return &DataStoreManager{
		Database: db,
		Endpoint: endpoint,
	}
}
