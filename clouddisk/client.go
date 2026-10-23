package clouddisk

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList      = "/api/v1/cloudDisk/list"
	pathPushFiles = "/api/v1/cloudDisk/pushFiles"
	pathDelete    = "/api/v1/cloudDisk/delFiles"
	pathSignedURL = "/api/v1/cloudDisk/signedUrl"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

// ListRequest allows at most 100 items per page.
type ListRequest struct {
	Keyword string `json:"keyword,omitempty"`
	common.PaginationRequest
}

type File struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	OriginalFileName string `json:"original_file_name"`
}

type ListResponse struct {
	List   []File `json:"list"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
	common.Pagination
}

type PushFilesRequest struct {
	// IDs contains at most 20 cloud drive file IDs.
	IDs []string `json:"ids"`
	// ImageIDs contains at most 20 target cloud phone IDs.
	ImageIDs []string `json:"image_ids"`
	// DestDir is the required destination directory, such as /sdcard/Download.
	DestDir string `json:"dest_dir"`
}

type PushFilesSuccess struct {
	ImageID string `json:"image_id"`
	ID      string `json:"id"`
}

type PushFilesFailure struct {
	ImageID string `json:"image_id"`
	ID      string `json:"id"`
	Err     string `json:"err"`
}

// PushFilesResponse can contain Fail entries even when err is nil.
type PushFilesResponse struct {
	Message string             `json:"message"`
	Success []PushFilesSuccess `json:"success"`
	Fail    []PushFilesFailure `json:"fail"`
}

type DeleteRequest struct {
	IDs []string `json:"ids"`
}

type SignedURLRequest struct {
	// Name is required and includes the filename extension.
	Name string `json:"name"`
	// IsApp is 1 for an application upload; zero omits the field.
	IsApp int `json:"is_app,omitempty"`
	// Pkg is required when IsApp is 1.
	Pkg string `json:"pkg,omitempty"`
}

// SignedURLResponse includes temporary upload credentials in SignedURL and Headers.
type SignedURLResponse struct {
	Method    string `json:"method"`
	SignedURL string `json:"signedUrl"`
	// Forward to storage, including the callback headers.
	Headers          map[string]string `json:"headers"`
	Name             string            `json:"name"`
	OriginalFileName string            `json:"original_file_name"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) PushFiles(ctx context.Context, req PushFilesRequest) (*PushFilesResponse, error) {
	var resp PushFilesResponse
	if err := c.core.Do(ctx, pathPushFiles, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Delete(ctx context.Context, ids []string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathDelete, DeleteRequest{IDs: ids}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SignedURL(ctx context.Context, req SignedURLRequest) (*SignedURLResponse, error) {
	var resp SignedURLResponse
	if err := c.core.Do(ctx, pathSignedURL, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Upload sends exactly size bytes from body. The storage request has no API key
// and no HTTP client timeout; set a context deadline.
func (c *Client) Upload(ctx context.Context, req SignedURLRequest, body io.Reader, size int64) (*SignedURLResponse, error) {
	signed, err := c.SignedURL(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := c.core.Upload(ctx, signed.Method, signed.SignedURL, signed.Headers, body, size); err != nil {
		return nil, err
	}
	return signed, nil
}

// UploadFile defaults req.Name to the file's base name. Set a context deadline for the upload.
func (c *Client) UploadFile(ctx context.Context, path string, req SignedURLRequest) (*SignedURLResponse, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("upload: %s is a directory", path)
	}
	if req.Name == "" {
		req.Name = filepath.Base(path)
	}
	return c.Upload(ctx, req, f, info.Size())
}
