package group

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathList   = "/api/v1/cloudPhone/groupList"
	pathAdd    = "/api/v1/cloudPhone/addToGroup"
	pathMove   = "/api/v1/cloudPhone/moveToGroup"
	pathCreate = "/api/v1/cloudPhone/createGroup"
	pathUpdate = "/api/v1/cloudPhone/updateGroup"
	pathDelete = "/api/v1/cloudPhone/deleteGroup"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

type ListRequest struct {
	Page int `json:"page,omitempty"`
}

type Item struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Remark string `json:"remark"`
}

type ListResponse struct {
	List []Item `json:"list"`
	common.Pagination
}

type GroupImagesRequest struct {
	ID       string   `json:"id"`
	ImageIDs []string `json:"image_ids"`
}

type CreateItem struct {
	Name   string `json:"name"`
	Sort   *int   `json:"sort,omitempty"`
	Remark string `json:"remark,omitempty"`
}

type UpdateItem struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Sort   *int    `json:"sort,omitempty"`
	Remark *string `json:"remark,omitempty"`
}

type CreateRequest struct {
	List []CreateItem `json:"list"`
}

type UpdateRequest struct {
	List []UpdateItem `json:"list"`
}

type MutationSuccess struct {
	Index  int    `json:"index"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Sort   int    `json:"sort"`
	Remark string `json:"remark"`
}

type MutationFailure struct {
	Index   int    `json:"index"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type MutationResponse struct {
	Success []MutationSuccess `json:"success"`
	Fail    []MutationFailure `json:"fail"`
}

type IDsRequest struct {
	IDs []string `json:"ids"`
}

func (c *Client) List(ctx context.Context, req ListRequest) (*ListResponse, error) {
	var resp ListResponse
	if err := c.core.Do(ctx, pathList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AddPhones(ctx context.Context, groupID string, imageIDs []string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAdd, GroupImagesRequest{ID: groupID, ImageIDs: imageIDs}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) MovePhones(ctx context.Context, groupID string, imageIDs []string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathMove, GroupImagesRequest{ID: groupID, ImageIDs: imageIDs}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Create(ctx context.Context, req CreateRequest) (*MutationResponse, error) {
	var resp MutationResponse
	if err := c.core.Do(ctx, pathCreate, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Update(ctx context.Context, req UpdateRequest) (*MutationResponse, error) {
	var resp MutationResponse
	if err := c.core.Do(ctx, pathUpdate, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Delete(ctx context.Context, ids []string) (*common.IDsOperationResponse, error) {
	var resp common.IDsOperationResponse
	if err := c.core.Do(ctx, pathDelete, IDsRequest{IDs: ids}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
