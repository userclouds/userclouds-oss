package s3shim

import "context"

// Controller is the interface for the s3shim controller
type Controller interface {
	CheckPermission(ctx context.Context, jwt string, path string) (bool, error)
	TransformData(ctx context.Context, data []byte) ([]byte, error)
}
