package app

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList          = "/api/v1/app/list"
	pathTeamList      = "/api/v1/app/teamList"
	pathInstall       = "/api/v1/app/install"
	pathInstalledList = "/api/v1/app/installedList"
	pathUninstall     = "/api/v1/app/uninstall"
	pathStart         = "/api/v1/app/start"
	pathStop          = "/api/v1/app/stop"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

type ListRequest struct {
	common.PaginationRequest
}

type Version struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Item struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Pkg         string    `json:"pkg"`
	VersionList []Version `json:"version_list"`
}

type ListResponse struct {
	List []Item `json:"list"`
	common.Pagination
}

type InstallRequest struct {
	ImageIDs     []string `json:"image_ids"`
	AppID        string   `json:"app_id"`
	AppVersionID string   `json:"app_version_id,omitempty"`
}

type InstalledRequest struct {
	ImageID string `json:"image_id"`
}

type InstalledResponse struct {
	List []string `json:"list"`
}

type PackageOperationRequest struct {
	ImageIDs []string `json:"image_ids"`
	Pkg      string   `json:"pkg"`
}

func (c *Client) ListPlatform(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListTeam(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathTeamList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Install(ctx context.Context, req InstallRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathInstall, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Installed(ctx context.Context, imageID string) (*InstalledResponse, error) {
	var resp InstalledResponse
	if err := c.core.Do(ctx, pathInstalledList, InstalledRequest{ImageID: imageID}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Uninstall(ctx context.Context, imageIDs []string, pkg string) (*common.MessageResponse, error) {
	return c.packageOperation(ctx, pathUninstall, imageIDs, pkg)
}

func (c *Client) Start(ctx context.Context, imageIDs []string, pkg string) (*common.MessageResponse, error) {
	return c.packageOperation(ctx, pathStart, imageIDs, pkg)
}

func (c *Client) Stop(ctx context.Context, imageIDs []string, pkg string) (*common.MessageResponse, error) {
	return c.packageOperation(ctx, pathStop, imageIDs, pkg)
}

func (c *Client) packageOperation(ctx context.Context, path string, imageIDs []string, pkg string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, path, PackageOperationRequest{ImageIDs: imageIDs, Pkg: pkg}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
