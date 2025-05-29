package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/takaishi/acb/internal/aws"
)

// URIからストレージ情報を解析する
type storageInfo struct {
	storageType string // "s3" または "file"
	bucket      string // S3の場合のバケット名
	path        string // ファイルパス
}

func parseStorageURI(uri string) (*storageInfo, error) {
	if uri == "" {
		return nil, fmt.Errorf("URI is not specified")
	}

	if strings.HasPrefix(uri, "s3://") {
		// S3 URI (s3://bucket/prefix/file.tar.gz)
		uri = strings.TrimPrefix(uri, "s3://")
		parts := strings.SplitN(uri, "/", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid S3 URI format: %s", uri)
		}
		bucket := parts[0]
		key := parts[1]

		return &storageInfo{
			storageType: "s3",
			bucket:      bucket,
			path:        key,
		}, nil
	} else if strings.HasPrefix(uri, "file://") {
		// File URI (file:///path/to/file.tar.gz)
		path := strings.TrimPrefix(uri, "file://")
		return &storageInfo{
			storageType: "file",
			path:        path,
		}, nil
	}

	return nil, fmt.Errorf("invalid URI format: %s", uri)
}

func readDataKey(info *storageInfo) ([]byte, error) {
	if info.storageType == "s3" {
		client, err := aws.NewS3Client(context.TODO())
		if err != nil {
			return nil, err
		}
		encryptedDataKey, err := client.GetObject(context.TODO(), info.bucket, info.path)
		if err != nil {
			return nil, err
		}
		return encryptedDataKey, nil
	} else if info.storageType == "file" {
		data, err := os.ReadFile(info.path)
		if err != nil {
			return nil, fmt.Errorf("failed to read data key file: %w", err)
		}

		return data, nil
	} else {
		return nil, fmt.Errorf("invalid storage type: %s", info.storageType)
	}
}
