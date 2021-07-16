// Package asana is a client for Asana API.
package asana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	libraryVersion = "0.1"
	userAgent      = "go-asana/" + libraryVersion
	defaultBaseURL = "https://app.asana.com/api/1.0/"
)

var defaultOptFields = map[string][]string{
	"tags":       {"name", "color", "notes"},
	"users":      {"name", "email", "photo"},
	"projects":   {"name", "color", "archived"},
	"workspaces": {"name", "is_organization"},
	"tasks":      {"name", "assignee", "assignee_status", "completed", "parent"},
}

var (
	// ErrUnauthorized can be returned on any call on response status code 401.
	ErrUnauthorized = errors.New("asana: unauthorized")
)

type (
	// Doer interface used for doing http calls.
	// Use it as point of setting Auth header or custom status code error handling.
	Doer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	// DoerFunc implements Doer interface.
	// Allow to transform any appropriate function "f" to Doer instance: DoerFunc(f).
	DoerFunc func(req *http.Request) (resp *http.Response, err error)

	Client struct {
		doer      Doer
		BaseURL   *url.URL
		UserAgent string
	}

	Workspace struct {
		ID           string `json:"gid,omitempty"`
		Name         string `json:"name,omitempty"`
		Organization bool   `json:"is_organization,omitempty"`
	}

	User struct {
		ID         string            `json:"gid,omitempty"`
		Email      string            `json:"email,omitempty"`
		Name       string            `json:"name,omitempty"`
		Photo      map[string]string `json:"photo,omitempty"`
		Workspaces []Workspace       `json:"workspaces,omitempty"`
	}

	Project struct {
		ID       string `json:"gid,omitempty"`
		Name     string `json:"name,omitempty"`
		Archived bool   `json:"archived,omitempty"`
		Color    string `json:"color,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	Task struct {
		ID             string    `json:"gid,omitempty"`
		Assignee       *User     `json:"assignee,omitempty"`
		AssigneeStatus string    `json:"assignee_status,omitempty"`
		CreatedAt      time.Time `json:"created_at,omitempty"`
		CreatedBy      User      `json:"created_by,omitempty"` // Undocumented field, but it can be included.
		Completed      bool      `json:"completed,omitempty"`
		Name           string    `json:"name,omitempty"`
		Hearts         []Heart   `json:"hearts,omitempty"`
		Notes          string    `json:"notes,omitempty"`
		ParentTask     *Task     `json:"parent,omitempty"`
		Projects       []Project `json:"projects,omitempty"`
		DueOn          string    `json:"due_on,omitempty"`
		DueAt          string    `json:"due_at,omitempty"`
	}
	// TaskUpdate is used to update a task.
	TaskUpdate struct {
		Notes   *string `json:"notes,omitempty"`
		Hearted *bool   `json:"hearted,omitempty"`
	}

	Story struct {
		ID        string    `json:"gid,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		CreatedBy User      `json:"created_by,omitempty"`
		Hearts    []Heart   `json:"hearts,omitempty"`
		Text      string    `json:"text,omitempty"`
		Type      string    `json:"type,omitempty"` // E.g., "comment", "system".
	}

	// Heart represents a ♥ action by a user.
	Heart struct {
		ID   string `json:"gid,omitempty"`
		User User   `json:"user,omitempty"`
	}

	Tag struct {
		ID    string `json:"gid,omitempty"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color,omitempty"`
		Notes string `json:"notes,omitempty"`
	}

	Filter struct {
		Archived       bool     `url:"archived,omitempty"`
		Assignee       string   `url:"assignee,omitempty"`
		Project        string   `url:"project,omitempty"`
		Workspace      string   `url:"workspace,omitempty"`
		CompletedSince string   `url:"completed_since,omitempty"`
		ModifiedSince  string   `url:"modified_since,omitempty"`
		OptFields      []string `url:"opt_fields,comma,omitempty"`
		OptExpand      []string `url:"opt_expand,comma,omitempty"`
	}

	request struct {
		Data interface{} `json:"data,omitempty"`
	}

	Response struct {
		Data   interface{} `json:"data,omitempty"`
		Errors Errors      `json:"errors,omitempty"`
	}

	Error struct {
		Phrase  string `json:"phrase,omitempty"`
		Message string `json:"message,omitempty"`
	}

	// Errors always has at least 1 element when returned.
	Errors []Error
)

func (f DoerFunc) Do(req *http.Request) (resp *http.Response, err error) {
	return f(req)
}

func (e Error) Error() string {
	return fmt.Sprintf("%v - %v", e.Message, e.Phrase)
}

func (e Errors) Error() string {
	var sErrs []string
	for _, err := range e {
		sErrs = append(sErrs, err.Error())
	}
	return strings.Join(sErrs, ", ")
}

// NewClient created new asana client with doer.
// If doer is nil then http.DefaultClient used intead.
func NewClient(doer Doer) *Client {
	if doer == nil {
		doer = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	client := &Client{doer: doer, BaseURL: baseURL, UserAgent: userAgent}
	return client
}

func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	workspaces := new([]Workspace)
	err := c.Request(ctx, "workspaces", nil, workspaces)
	return *workspaces, err
}

func (c *Client) ListUsers(ctx context.Context, opt *Filter) ([]User, error) {
	users := new([]User)
	err := c.Request(ctx, "users", opt, users)
	return *users, err
}

func (c *Client) ListProjects(ctx context.Context, opt *Filter) ([]Project, error) {
	projects := new([]Project)
	err := c.Request(ctx, "projects", opt, projects)
	return *projects, err
}

func (c *Client) ListTasks(ctx context.Context, opt *Filter) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request(ctx, "tasks", opt, tasks)
	return *tasks, err
}

func (c *Client) GetTask(ctx context.Context, id string, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(ctx, fmt.Sprintf("tasks/%s", id), opt, task)
	return *task, err
}

// UpdateTask updates a task.
//
// https://asana.com/developers/api-reference/tasks#update
func (c *Client) UpdateTask(ctx context.Context, id string, tu TaskUpdate, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.request(ctx, "PUT", fmt.Sprintf("tasks/%s", id), tu, nil, opt, task)
	return *task, err
}

// CreateTask creates a task.
//
// https://asana.com/developers/api-reference/tasks#create
func (c *Client) CreateTask(ctx context.Context, fields map[string]string, opts *Filter) (Task, error) {
	task := new(Task)
	err := c.request(ctx, "POST", "tasks", nil, toURLValues(fields), opts, task)
	return *task, err
}

func (c *Client) ListProjectTasks(ctx context.Context, projectID string, opt *Filter) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request(ctx, fmt.Sprintf("projects/%s/tasks", projectID), opt, tasks)
	return *tasks, err
}

func (c *Client) ListTaskStories(ctx context.Context, taskID string, opt *Filter) ([]Story, error) {
	stories := new([]Story)
	err := c.Request(ctx, fmt.Sprintf("tasks/%s/stories", taskID), opt, stories)
	return *stories, err
}

func (c *Client) ListTags(ctx context.Context, opt *Filter) ([]Tag, error) {
	tags := new([]Tag)
	err := c.Request(ctx, "tags", opt, tags)
	return *tags, err
}

func (c *Client) GetAuthenticatedUser(ctx context.Context, opt *Filter) (User, error) {
	user := new(User)
	err := c.Request(ctx, "users/me", opt, user)
	return *user, err
}

func (c *Client) GetUserByID(ctx context.Context, id string, opt *Filter) (User, error) {
	user := new(User)
	err := c.Request(ctx, fmt.Sprintf("users/%s", id), opt, user)
	return *user, err
}

func (c *Client) Request(ctx context.Context, path string, opt *Filter, v interface{}) error {
	return c.request(ctx, "GET", path, nil, nil, opt, v)
}

// request makes a request to Asana API, using method, at path, sending data or form with opt filter.
// Only data or form could be sent at the same time. If both provided form will be omitted.
// Also it's possible to do request with nil data and form.
// The response is populated into v, and any error is returned.
func (c *Client) request(ctx context.Context, method string, path string, data interface{}, form url.Values, opt *Filter, v interface{}) error {
	if opt == nil {
		opt = &Filter{}
	}
	if len(opt.OptFields) == 0 {
		// We should not modify opt provided to Request.
		newOpt := *opt
		opt = &newOpt
		opt.OptFields = defaultOptFields[path]
	}
	urlStr, err := addOptions(path, opt)
	if err != nil {
		return err
	}
	rel, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	u := c.BaseURL.ResolveReference(rel)
	var body io.Reader
	if data != nil {
		b, err := json.Marshal(request{Data: data})
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	} else if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	} else if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.doer.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	res := &Response{Data: v}
	err = json.NewDecoder(resp.Body).Decode(res)
	if len(res.Errors) > 0 {
		return res.Errors
	}
	return err
}

func addOptions(s string, opt interface{}) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}
	u.RawQuery = qs.Encode()
	return u.String(), nil
}

func toURLValues(m map[string]string) url.Values {
	values := make(url.Values)
	for k, v := range m {
		values[k] = []string{v}
	}
	return values
}
