package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"net/textproto"
	"netops/conf"
	"strconv"
	"strings"
)

// IssueType 工单类型
type IssueType struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Value 工单属地
type Value struct {
	Value string `json:"value"`
}

// UserInfo 工单用户
type UserInfo struct {
	Name        string `json:"name"`         // 用户名
	DisplayName string `json:"displayName"`  // 中文名
	Email       string `json:"emailAddress"` // 邮箱地址
}

// Attachment 工单附件数据格式
type Attachment struct {
	Id       string `json:"id"`       // 附件ID
	FileName string `json:"filename"` // 附件名
	Created  string `json:"created"`  // 创建时间
}

func (a Attachment) String() string {
	return fmt.Sprintf("filename: %s, created: %s", a.FileName, a.Created)
}

// AttachmentResult 工单附件返回数据格式
type AttachmentResult struct {
	Self     string `json:"self"`     // 本附件API URL
	Filename string `json:"filename"` // 文件名
	Author   struct {
		Self        string `json:"self"` // 用户信息API
		Key         string `json:"key"`
		Name        string `json:"name"`        // 用户名
		DisplayName string `json:"displayName"` // 中文名称
	}
	Created  string `json:"created"`  // 创建时间
	Size     int64  `json:"size"`     // 文件大小
	MimeType string `json:"mimeType"` // 文件类型
	Content  string `json:"content"`  // 文件下载地址
}

func (a AttachmentResult) String() string {
	return fmt.Sprintf("self: %s, filename: %s, created: %s, size: %d, content: %s", a.Self, a.Filename,
		a.Created, a.Size, a.Content)
}

// Status 工单状态数据格式
type Status struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// Project 工单项目数据格式
type Project struct {
	Id   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Department struct {
	Value string `json:"value"`
	Child Value  `json:"child"`
}

// Fields 工单fields
type Fields struct {
	IssueType           IssueType    `json:"issuetype"`         // 工单类型
	Region              Value        `json:"customfield_10816"` // 变更属地
	Environment         Value        `json:"customfield_10817"` // 变更环境
	ImplementContent    Value        `json:"customfield_10850"` // 实施内容
	Department          Department   `json:"customfield_15124"` // 部门
	ExpireTime          Value        `json:"customfield_15005"` // 工单过期时间
	NetworkOpeningRange Value        `json:"customfield_14465"` // 网络开通范围
	Creator             UserInfo     `json:"creator"`           // 用户类型
	Description         string       `json:"description"`       // 描述
	CreateTime          string       `json:"created"`           // 创建时间
	UpdateTime          string       `json:"updated"`           // 更新时间
	Attachment          []Attachment `json:"attachment"`        // 附件列表
	Assignee            UserInfo     `json:"assignee"`          // 经办人
	Status              Status       `json:"status"`            // 状态
	Project             Project      `json:"project"`           // 项目名称
	Summary             string       `json:"summary"`           // 概要
}

func (f Fields) String() string {
	return fmt.Sprintf("type: %s,  creator: %s, createtime: %s, "+
		"updatetime: %s, attachment: %s, status: %s, project: %s", f.IssueType.Name,
		f.Creator.DisplayName, f.CreateTime, f.UpdateTime, f.Attachment, f.Status.Name, f.Project.Name)
}

// Issue 单个工单数据格式
type Issue struct {
	Id     string `json:"id"`
	Key    string `json:"key"`
	Fields Fields `json:"fields"`
}

func (i Issue) String() string {
	return i.Key
}

// SearchIssue 查询工单数据结构
type SearchIssue struct {
	Expand     string   `json:"expand"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Total      int      `json:"total"`
	Issues     []*Issue `json:"issues"`
}

// CreateIssueResult 创建工单返回结果
type CreateIssueResult struct {
	Id     string            `json:"id"`
	Key    string            `json:"key"`
	Self   string            `json:"self"`
	Errors map[string]string `json:"errors"`
}

// CreateIssueJson 创建工单数据结构
type CreateIssueJson struct {
	Fields map[string]interface{} `json:"fields"`
}

// TransitionResult 工单流程返回数据格式
type TransitionResult struct {
	Expand      string        `json:"expand"`
	Transitions []*Transition `json:"transitions"`
}
type Transition struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"to"`
}

func (t *Transition) String() string {
	return t.To.Name
}

// AssigneeResult 分配流程返回数据格式
type AssigneeResult struct {
	ErrorMessages interface{} `json:"errorMessages"`
	Errors        struct {
		Assignee string `json:"assignee"`
	} `json:"errors"`
}

type jiraHandler struct {
	HttpHandler
	username string
	password string
	host     string
	Err      error
}

func NewJiraHandler() *jiraHandler {
	jira := &jiraHandler{
		username: conf.Config.Jira.User,
		password: conf.Config.Jira.Password,
		host:     conf.Config.Jira.Server,
	}
	jira.init()
	jira.SetBasicAuth(jira.username, jira.password)
	jira.isAuthor()
	return jira
}

func (j *jiraHandler) GetUsername() string {
	return j.username
}

func (j *jiraHandler) newAuthorization(username, password string) string {
	str := fmt.Sprintf("%s:%s", username, password)
	input := []byte(str)
	return base64.StdEncoding.EncodeToString(input)
}

// 验证用户信息
func (j *jiraHandler) isAuthor() {
	// 测试登录接口
	var result interface{}
	j.Err = j.Get("/rest/auth/1/session", nil, &result)
}

// GetIssue 根据工单号, 获取工单信息
func (j *jiraHandler) GetIssue(key string) (*Issue, error) {
	if j.Err != nil {
		return nil, j.Err
	}
	issue := Issue{}
	if e := j.Get(fmt.Sprintf("/rest/api/2/issue/%s", key), nil, &issue); e != nil {
		return nil, fmt.Errorf("获取工单失败, 工单号: %s, err: %w", key, e)
	}
	return &issue, nil
}

// SearchIssues 获取符合查询条件的工单信息, jql查询语句
func (j *jiraHandler) SearchIssues(jql string) ([]*Issue, error) {
	if j.Err != nil {
		return nil, j.Err
	}
	searchIssue := SearchIssue{}
	if e := j.Get("/rest/api/2/search", map[string]string{"jql": jql}, &searchIssue); e != nil {
		return nil, e
	}
	return searchIssue.Issues, nil
}

// GetAttachment 获取工单附件地址
func (j *jiraHandler) GetAttachment(attachmentId string) (*AttachmentResult, error) {
	if j.Err != nil {
		return nil, j.Err
	}
	uri := fmt.Sprintf("/rest/api/2/attachment/%s", attachmentId)
	result := AttachmentResult{}
	if e := j.Get(uri, nil, &result); e != nil {
		return nil, fmt.Errorf("获取工单附件失败, 附件ID: %s, err: %w", attachmentId, e)
	}
	return &result, nil
}

// ReadAttachment 读取附件内容
func (j *jiraHandler) ReadAttachment(url string) ([]byte, error) {
	if j.Err != nil {
		return nil, j.Err
	}
	result := make([]byte, 0)
	if e := j.Get(url, nil, &result); e != nil {
		return nil, fmt.Errorf("读取附件失败, err: %w", e)
	}
	return result, nil
}

// UpdateAssignee 修改工单经办人
func (j *jiraHandler) UpdateAssignee(key, assignee string) error {
	if j.Err != nil {
		return j.Err
	}
	url := fmt.Sprintf("/rest/api/latest/issue/%s/assignee", key)
	data := map[string]string{"name": assignee}
	assigneeResult := AssigneeResult{}
	if e := j.Put(url, &data, assigneeResult); e != nil {
		return fmt.Errorf("修改经办人失败,工单号: %s, 经办人: %s, err: %w", key, assignee, e)
	}
	if assigneeResult.Errors.Assignee != "" {
		return fmt.Errorf("修改工单经办人异常, 工单号: %s, err: %s", key, assigneeResult.Errors.Assignee)
	}
	return nil
}

// GetTransition 获取工单所有状态
func (j *jiraHandler) GetTransition(key string) ([]*Transition, error) {
	if j.Err != nil {
		return nil, j.Err
	}
	url := fmt.Sprintf("/rest/api/2/issue/%s/transitions", key)
	transitionsResult := TransitionResult{}
	if e := j.Get(url, nil, &transitionsResult); e != nil {
		return nil, fmt.Errorf("获取工单状态失败, 工单号: %s, err: %w", key, e)
	}
	return transitionsResult.Transitions, nil
}

// GetTransitionId 获取工单状态ID
func (j *jiraHandler) GetTransitionId(key, transition string) (int, error) {
	if j.Err != nil {
		return 0, j.Err
	}
	transitions, e := j.GetTransition(key)
	if e != nil {
		return 0, e
	}
	for _, v := range transitions {
		if v.Name == transition {
			transitionId, _ := strconv.Atoi(v.Id)
			return transitionId, nil
		}
	}
	return 0, fmt.Errorf("未找到工单流程, 工单号: %s, 流程名: %s", key, transition)
}

// UpdateTransition 更新工单流程
func (j *jiraHandler) UpdateTransition(key string, transitionId int) error {
	if j.Err != nil {
		return j.Err
	}
	url := fmt.Sprintf("/rest/api/2/issue/%s/transitions", key)
	// 登录用户是网络自动化平台，并且经办人不是网络自动化平台，将经办人改为网络自动化平台
	data := map[string]map[string]int{
		"transition": {"id": transitionId},
	}
	result := struct {
		ErrorMessages []string    `json:"errorMessages"`
		Errors        interface{} `json:"errors"`
	}{}
	if e := j.Post(url, &data, &result); e != nil {
		return fmt.Errorf("更新工单流程失败, err: %s", strings.Join(result.ErrorMessages, ","))
	}
	return nil
}

// AddComment 添加评论
func (j *jiraHandler) AddComment(key, comment string) error {
	url := fmt.Sprintf("/rest/api/2/issue/%s/comment", key)
	data := map[string]string{
		"body": comment,
	}
	if e := j.Post(url, &data, nil); e != nil {
		return fmt.Errorf("添加评论失败, err: %w", e)
	}
	return nil
}

// AddAttachment 添加附件
func (j *jiraHandler) AddAttachment(key, filename string, body []byte) error {
	url := fmt.Sprintf("/rest/api/2/issue/%s/attachments", key)
	buffer := new(bytes.Buffer)
	w := multipart.NewWriter(buffer)

	// 取出内容类型
	contentType := w.FormDataContentType()

	// 将文件数据写入
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%s; filename="%s"`, "file", filename))

	switch {
	case strings.HasSuffix(filename, ".xlsx"):
		h.Set("Content-Type", "application/vnd.ms-excel")
	case strings.HasSuffix(filename, ".txt"):
		h.Set("Content-Type", "text/plain")
	}

	pa, _ := w.CreatePart(h)
	_, err := pa.Write(body)
	if err != nil {
		return fmt.Errorf("写入附件数据失败, err: %w", err)
	}
	_ = w.Close()
	headers := make(map[string]string)
	headers["Content-Type"] = contentType
	headers["X-Atlassian-Token"] = "nocheck"
	headers["Accept"] = "application/json"
	headers["Connection"] = "keep-alive"
	j.SetHeaders(headers)
	if e := j.Post(url, buffer, nil); e != nil {
		return fmt.Errorf("添加附件失败, err: %w", e)
	}
	return nil
}
