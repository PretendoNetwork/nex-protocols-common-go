package main

import (
	"fmt"
	"net/url"
	"time"

	common_globals "github.com/PretendoNetwork/nex-protocols-common-go/v2/globals"
)

// * The tests never upload anything, they just call CompletePostObject
// * as if the upload happened
const (
	s3Bucket  = "test"
	s3KeyBase = "datastore"
	s3Host    = "https://127.0.0.1:9000"
)

// stubS3Manager is an S3Manager which pretends every request
// succeeded, so that the methods which use the file server can be
// tested without running a storage server
type stubS3Manager struct{}

func (s stubS3Manager) PresignGetObject(bucket, key string, lifetime time.Duration) (*common_globals.S3GetObjectData, error) {
	objectURL, err := url.Parse(fmt.Sprintf("%s/%s/%s", s3Host, bucket, key))
	if err != nil {
		return nil, err
	}

	return &common_globals.S3GetObjectData{
		URL:            objectURL,
		RequestHeaders: map[string]string{},
		RootCACert:     []byte{},
		Size:           0,
	}, nil
}

func (s stubS3Manager) PresignPostObject(bucket, key string, lifetime time.Duration) (*common_globals.S3PostObjectData, error) {
	objectURL, err := url.Parse(fmt.Sprintf("%s/%s/%s", s3Host, bucket, key))
	if err != nil {
		return nil, err
	}

	return &common_globals.S3PostObjectData{
		URL:            objectURL,
		FormData:       map[string]string{"key": key},
		RequestHeaders: map[string]string{},
		RootCACert:     []byte{},
	}, nil
}

func (s stubS3Manager) PutObject(bucket, key, data string) error {
	return nil
}
