package automation

import (
	"context"

	"github.com/wyosa/duoplus-go-sdk/common"
	"github.com/wyosa/duoplus-go-sdk/internal/clientcore"
)

const (
	pathUserTemplateList     = "/api/v1/automation/userTemplateList"
	pathOfficialTemplateList = "/api/v1/automation/officialTemplateList"
	pathAddTask              = "/api/v1/automation/addTask"
	pathAddPlan              = "/api/v1/automation/addPlan"
	pathPlanList             = "/api/v1/automation/planList"
	pathSavePlan             = "/api/v1/automation/savePlan"
	pathSetPlanStatus        = "/api/v1/automation/setPlanStatus"
	pathDeletePlan           = "/api/v1/automation/deletePlan"
	pathTaskList             = "/api/v1/automation/taskList"
	pathTaskLogList          = "/api/v1/automation/taskLogList"
	pathUpdateTaskTime       = "/api/v1/automation/updateTaskTime"
	pathSetTaskStatus        = "/api/v1/automation/setTaskStatus"
	pathUserTemplateInfo     = "/api/v1/automation/userTemplateInfo"
	pathOfficialTemplateInfo = "/api/v1/automation/officialTemplateInfo"
)

type Client struct {
	core *clientcore.Client
}

func New(core *clientcore.Client) *Client {
	return &Client{core: core}
}

type TemplateListRequest struct {
	Name string `json:"name,omitempty"`
	common.PaginationRequest
}

type Template struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

type TemplateListResponse struct {
	List []Template `json:"list"`
	common.Pagination
}

type OfficialTemplate struct {
	Template
	Version           []string `json:"version"`
	Lang              []string `json:"lang"`
	RPAAndroidVersion string   `json:"rpa_android_version"`
	IsAIRPA           int      `json:"is_ai_rpa"`
	IsCloudDisk       int      `json:"is_cloud_disk"`
	IsPinned          bool     `json:"is_pinned"`
	PinnedAt          string   `json:"pinned_at"`
}

type OfficialTemplateListResponse struct {
	List []OfficialTemplate `json:"list"`
	common.Pagination
}

type TemplateInfoRequest struct {
	ID string `json:"id"`
}

type TemplateConfigItem struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Desc     string `json:"desc"`
}

type TemplateDetail struct {
	ID            string               `json:"id"`
	AppID         string               `json:"app_id"`
	Name          string               `json:"name"`
	Desc          string               `json:"desc"`
	ExceptionType int                  `json:"exception_type"`
	DumpType      int                  `json:"dump_type"`
	Config        []TemplateConfigItem `json:"config"`
}

type templateInfoResponse struct {
	Info TemplateDetail `json:"info"`
}

type ConfigValue struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type TaskImage struct {
	ImageID        string                 `json:"image_id"`
	Config         map[string]ConfigValue `json:"config,omitempty"`
	IssueAt        string                 `json:"issue_at,omitempty"`
	StartAt        string                 `json:"start_at,omitempty"`
	EndAt          string                 `json:"end_at,omitempty"`
	ExecuteType    int                    `json:"execute_type,omitempty"`
	GapTime        int                    `json:"gap_time,omitempty"`
	ExecuteTime    string                 `json:"execute_time,omitempty"`
	ExecuteEndTime string                 `json:"execute_end_time,omitempty"`
	Mode           int                    `json:"mode,omitempty"`
	Weeks          []string               `json:"weeks,omitempty"`
	Days           []string               `json:"days,omitempty"`
}

type CreateTaskRequest struct {
	TemplateID   string      `json:"template_id"`
	TemplateType int         `json:"template_type"`
	Name         string      `json:"name"`
	Remark       string      `json:"remark,omitempty"`
	Images       []TaskImage `json:"images"`
}

type PlanListRequest struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Status       []int  `json:"status,omitempty"`
	TemplateType []int  `json:"template_type,omitempty"`
	Remark       string `json:"remark,omitempty"`
	common.PaginationRequest
}

type Plan struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Remark       string `json:"remark"`
	TaskTypeName string `json:"task_type_name"`
	Status       int    `json:"status"`
	CreatedAt    string `json:"created_at"`
}

type PlanListResponse struct {
	List []Plan `json:"list"`
	common.Pagination
}

type SavePlanRequest struct {
	ID     string      `json:"id"`
	Name   string      `json:"name,omitempty"`
	Remark string      `json:"remark,omitempty"`
	Images []TaskImage `json:"images"`
}

type SetPlanStatusRequest struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

type DeletePlanRequest struct {
	ID string `json:"id"`
}

type TaskListRequest struct {
	ID               string `json:"id,omitempty"`
	IssueAtStart     string `json:"issue_at_start"`
	IssueAtEnd       string `json:"issue_at_end"`
	Status           []int  `json:"status,omitempty"`
	TemplateType     []int  `json:"template_type,omitempty"`
	Name             string `json:"name,omitempty"`
	ImageName        string `json:"image_name,omitempty"`
	ImageIP          string `json:"image_ip,omitempty"`
	ExecutionAtStart string `json:"execution_at_start,omitempty"`
	ExecutionAtEnd   string `json:"execution_at_end,omitempty"`
	SortBy           string `json:"sort_by,omitempty"`
	Order            string `json:"order,omitempty"`
	common.PaginationRequest
}

type Task struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	TaskTypeName  string `json:"task_type_name"`
	ImageName     string `json:"image_name"`
	IP            string `json:"ip"`
	Remark        string `json:"remark"`
	Status        int    `json:"status"`
	IssueAt       string `json:"issue_at"`
	StartAt       string `json:"start_at"`
	FinishAt      string `json:"finish_at"`
	CostTime      int    `json:"cost_time"`
	ExecutionTime string `json:"execution_time"`
	CreatedAt     string `json:"created_at"`
}

type TaskListResponse struct {
	List []Task `json:"list"`
	common.Pagination
}

type TaskLogListRequest struct {
	TaskID   string `json:"task_id"`
	CursorID string `json:"cursor_id,omitempty"`
}

type TaskLogResultInfo struct {
	Action                 string         `json:"action"`
	Result                 bool           `json:"result"`
	ErrorMessage           string         `json:"error_message"`
	ErrorImg               string         `json:"error_img"`
	ConditionResultMessage string         `json:"condition_result_message"`
	ExtraData              map[string]any `json:"extra_data"`
}

type TaskLog struct {
	ID         string            `json:"id"`
	ResultInfo TaskLogResultInfo `json:"result_info"`
	StartAt    string            `json:"start_at"`
	FinishAt   string            `json:"finish_at"`
	CreatedAt  string            `json:"created_at"`
}

type TaskLogListResponse struct {
	List []TaskLog `json:"list"`
}

type UpdateTaskTimeRequest struct {
	ID      string `json:"id"`
	IssueAt string `json:"issue_at"`
}

type SetTaskStatusRequest struct {
	IDs    []string `json:"ids"`
	Status int      `json:"status"`
}

func (c *Client) ListUserTemplates(ctx context.Context, req TemplateListRequest) (*TemplateListResponse, error) {
	var resp TemplateListResponse
	if err := c.core.Do(ctx, pathUserTemplateList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListOfficialTemplates(ctx context.Context, req TemplateListRequest) (*OfficialTemplateListResponse, error) {
	var resp OfficialTemplateListResponse
	if err := c.core.Do(ctx, pathOfficialTemplateList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UserTemplateInfo(ctx context.Context, id string) (*TemplateDetail, error) {
	return c.templateInfo(ctx, pathUserTemplateInfo, id)
}

func (c *Client) OfficialTemplateInfo(ctx context.Context, id string) (*TemplateDetail, error) {
	return c.templateInfo(ctx, pathOfficialTemplateInfo, id)
}

func (c *Client) templateInfo(ctx context.Context, path, id string) (*TemplateDetail, error) {
	var resp templateInfoResponse
	if err := c.core.Do(ctx, path, TemplateInfoRequest{ID: id}, &resp); err != nil {
		return nil, err
	}
	return &resp.Info, nil
}

func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathAddTask, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreatePlan(ctx context.Context, req CreateTaskRequest) (*common.IDResponse, error) {
	var resp common.IDResponse
	if err := c.core.Do(ctx, pathAddPlan, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListPlans(ctx context.Context, req PlanListRequest) (*PlanListResponse, error) {
	var resp PlanListResponse
	if err := c.core.Do(ctx, pathPlanList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SavePlan(ctx context.Context, req SavePlanRequest) (*common.IDResponse, error) {
	var resp common.IDResponse
	if err := c.core.Do(ctx, pathSavePlan, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetPlanStatus(ctx context.Context, req SetPlanStatusRequest) (*common.IDResponse, error) {
	var resp common.IDResponse
	if err := c.core.Do(ctx, pathSetPlanStatus, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) DeletePlan(ctx context.Context, id string) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathDeletePlan, DeletePlanRequest{ID: id}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListTasks(ctx context.Context, req TaskListRequest) (*TaskListResponse, error) {
	var resp TaskListResponse
	if err := c.core.Do(ctx, pathTaskList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) ListTaskLogs(ctx context.Context, req TaskLogListRequest) (*TaskLogListResponse, error) {
	var resp TaskLogListResponse
	if err := c.core.Do(ctx, pathTaskLogList, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateTaskTime(ctx context.Context, req UpdateTaskTimeRequest) (*common.MessageResponse, error) {
	var resp common.MessageResponse
	if err := c.core.Do(ctx, pathUpdateTaskTime, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) SetTaskStatus(ctx context.Context, req SetTaskStatusRequest) (*common.OperationResult, error) {
	var resp common.OperationResult
	if err := c.core.Do(ctx, pathSetTaskStatus, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
