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

// TemplateListRequest allows at most 100 items per page.
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
	Key   string `json:"key"`
	Value any    `json:"value"`
	// Type is file, textarea, boolean, number, string, or excel.
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Desc     string `json:"desc"`
}

type TemplateDetail struct {
	ID    string `json:"id"`
	AppID string `json:"app_id"`
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	// ExceptionType is 0 to skip a failed action or 1 to abort.
	ExceptionType int `json:"exception_type"`
	// DumpType is 0 for default mode or 1 for accessibility mode.
	DumpType int                  `json:"dump_type"`
	Config   []TemplateConfigItem `json:"config"`
}

type templateInfoResponse struct {
	Info TemplateDetail `json:"info"`
}

type ConfigValue struct {
	Key string `json:"key"`
	// File variables take an array of IDs/URLs; boolean and number examples use strings.
	Value any `json:"value"`
	// Type is file, textarea, boolean, number, string, or excel.
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type TaskImage struct {
	ImageID string                 `json:"image_id"`
	Config  map[string]ConfigValue `json:"config,omitempty"`
	// Required for CreateTask. Format: 2006-01-02 15:04.
	IssueAt string `json:"issue_at,omitempty"`
	// StartAt and EndAt are required for plans. Format: 2006-01-02 15:04.
	StartAt string `json:"start_at,omitempty"`
	EndAt   string `json:"end_at,omitempty"`
	// ExecuteType is 1 (interval), 2 (daily), 3 (weekly), or 4 (monthly).
	ExecuteType int `json:"execute_type,omitempty"`
	// Required for interval execution; the API docs do not specify a unit.
	GapTime int `json:"gap_time,omitempty"`
	// ExecuteTime is required for daily, weekly, and monthly execution, in 15:04 format.
	ExecuteTime string `json:"execute_time,omitempty"`
	// ExecuteEndTime is required for frequency 2, 3, or 4 when Mode is 2; format: 15:04.
	ExecuteEndTime string `json:"execute_end_time,omitempty"`
	// Mode is 1 (single execution) or 2 (loop execution); required for frequency 2, 3, or 4.
	Mode int `json:"mode,omitempty"`
	// Required for weekly execution. Weeks/Days use strings in CN, numbers in EN edit examples.
	Weeks []string `json:"weeks,omitempty"`
	// Required for monthly execution.
	Days []string `json:"days,omitempty"`
}

type CreateTaskRequest struct {
	TemplateID string `json:"template_id"`
	// TemplateType is 1 for official templates or 2 for custom templates.
	TemplateType int         `json:"template_type"`
	Name         string      `json:"name"`
	Remark       string      `json:"remark,omitempty"`
	Images       []TaskImage `json:"images"`
}

// PlanListRequest allows at most 100 items per page.
type PlanListRequest struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// Status filters use Plan.Status values.
	Status []int `json:"status,omitempty"`
	// TemplateType filters use 1 for official or 2 for custom templates.
	TemplateType []int  `json:"template_type,omitempty"`
	Remark       string `json:"remark,omitempty"`
	common.PaginationRequest
}

type Plan struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Remark       string `json:"remark"`
	TaskTypeName string `json:"task_type_name"`
	// Status is 0 (not started), 1 (executing), 2 (paused), or 3 (finished).
	Status    int    `json:"status"`
	CreatedAt string `json:"created_at"`
}

type PlanListResponse struct {
	List []Plan `json:"list"`
	common.Pagination
}

// SavePlanRequest leaves nil Name and Remark unchanged; common.Ptr("") sends an empty value.
type SavePlanRequest struct {
	ID     string      `json:"id"`
	Name   *string     `json:"name,omitempty"`
	Remark *string     `json:"remark,omitempty"`
	Images []TaskImage `json:"images"`
}

type SetPlanStatusRequest struct {
	ID string `json:"id"`
	// Status is 0 to pause or 1 to execute; these are not Plan.Status values.
	Status int `json:"status"`
}

type DeletePlanRequest struct {
	ID string `json:"id"`
}

// TaskListRequest requires IssueAtStart and IssueAtEnd; at most 100 items per page.
type TaskListRequest struct {
	ID string `json:"id,omitempty"`
	// IssueAtStart and IssueAtEnd: 2006-01-02 15:04:05. Timezone unspecified by the API.
	IssueAtStart string `json:"issue_at_start"`
	IssueAtEnd   string `json:"issue_at_end"`
	// Status filters use Task.Status values.
	Status []int `json:"status,omitempty"`
	// TemplateType filters use 1 for official or 2 for custom templates.
	TemplateType []int  `json:"template_type,omitempty"`
	Name         string `json:"name,omitempty"`
	ImageName    string `json:"image_name,omitempty"`
	ImageIP      string `json:"image_ip,omitempty"`
	// ExecutionAtStart and ExecutionAtEnd: 2006-01-02 15:04:05.
	ExecutionAtStart string `json:"execution_at_start,omitempty"`
	ExecutionAtEnd   string `json:"execution_at_end,omitempty"`
	// SortBy is issue_at or created_at.
	SortBy string `json:"sort_by,omitempty"`
	// Order is asc or desc.
	Order string `json:"order,omitempty"`
	common.PaginationRequest
}

type Task struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TaskTypeName string `json:"task_type_name"`
	ImageName    string `json:"image_name"`
	IP           string `json:"ip"`
	Remark       string `json:"remark"`
	// Status is 0 (pending), 1 (executing), 2 (paused), 3 (finished), 4 (failed), or 5 (canceled).
	Status   int    `json:"status"`
	IssueAt  string `json:"issue_at"`
	StartAt  string `json:"start_at"`
	FinishAt string `json:"finish_at"`
	// CostTime is execution duration in seconds.
	CostTime      int    `json:"cost_time"`
	ExecutionTime string `json:"execution_time"`
	CreatedAt     string `json:"created_at"`
}

type TaskListResponse struct {
	List []Task `json:"list"`
	common.Pagination
}

type TaskLogListRequest struct {
	TaskID string `json:"task_id"`
	// Last log ID from the previous batch; omit for the first batch.
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

// TaskLogListResponse has an empty List when there are no more logs.
type TaskLogListResponse struct {
	List []TaskLog `json:"list"`
}

type UpdateTaskTimeRequest struct {
	ID string `json:"id"`
	// Format: 2006-01-02 15:04.
	IssueAt string `json:"issue_at"`
}

type SetTaskStatusRequest struct {
	IDs []string `json:"ids"`
	// Status is 0 to re-execute or 5 to cancel.
	Status int `json:"status"`
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

// UpdateTaskTime only applies to tasks waiting for execution.
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
