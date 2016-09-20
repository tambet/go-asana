// Package asana is a client for Asana API.
package asana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type (
	//Doer interface used for doing http calls.
	//Use it as point of setting Auth header or custom status code error handling.
	Doer interface {
		Do(req *http.Request) (*http.Response, error)
	}

	//DoerFunc implements Doer interface.
	//Allow to transform any appropriate function "f" to Doer instance: DoerFunc(f).
	DoerFunc func(req *http.Request) (resp *http.Response, err error)

	Client struct {
		doer      Doer
		BaseURL   *url.URL
		UserAgent string
	}

	Workspace struct {
		ID           int64  `json:"id,omitempty"`
		Name         string `json:"name,omitempty"`
		Organization bool   `json:"is_organization,omitempty"`
	}

	User struct {
		ID         int64             `json:"id,omitempty"`
		Email      string            `json:"email,omitempty"`
		Name       string            `json:"name,omitempty"`
		Photo      map[string]string `json:"photo,omitempty"`
		Workspaces []Workspace       `json:"workspaces,omitempty"`
	}

	Project struct {
		ID       int64  `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Archived bool   `json:"archived,omitempty"`
		Color    string `json:"color,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	Task struct {
		ID             int64     `json:"id,omitempty"`
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
	}
	// TaskUpdate is used to update a task.
	TaskUpdate struct {
		Notes   *string `json:"notes,omitempty"`
		Hearted *bool   `json:"hearted,omitempty"`
	}

	Story struct {
		ID        int64     `json:"id,omitempty"`
		CreatedAt time.Time `json:"created_at,omitempty"`
		CreatedBy User      `json:"created_by,omitempty"`
		Hearts    []Heart   `json:"hearts,omitempty"`
		Text      string    `json:"text,omitempty"`
		Type      string    `json:"type,omitempty"` // E.g., "comment", "system".
	}

	// Heart represents a ♥ action by a user.
	Heart struct {
		ID   int64 `json:"id,omitempty"`
		User User  `json:"user,omitempty"`
	}

	Tag struct {
		ID    int64  `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color,omitempty"`
		Notes string `json:"notes,omitempty"`
	}

	Filter struct {
		Archived       bool     `url:"archived,omitempty"`
		Assignee       int64    `url:"assignee,omitempty"`
		Project        int64    `url:"project,omitempty"`
		Workspace      int64    `url:"workspace,omitempty"`
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
		Errors []Error     `json:"errors,omitempty"`
	}

	Error struct {
		Phrase  string `json:"phrase,omitempty"`
		Message string `json:"message,omitempty"`
	}
)

func (f DoerFunc) Do(req *http.Request) (resp *http.Response, err error) {
	return f(req)
}

func (e Error) Error() string {
	return fmt.Sprintf("%v - %v", e.Message, e.Phrase)
}

//NewClient created new asana client with doer.
//If doer is nil then http.DefaultClient used intead.
func NewClient(doer Doer) *Client {
	if doer == nil {
		doer = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	client := &Client{doer: doer, BaseURL: baseURL, UserAgent: userAgent}
	return client
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	workspaces := new([]Workspace)
	err := c.Request("workspaces", nil, workspaces)
	return *workspaces, err
}

func (c *Client) ListUsers(opt *Filter) ([]User, error) {
	users := new([]User)
	err := c.Request("users", opt, users)
	return *users, err
}

func (c *Client) ListProjects(opt *Filter) ([]Project, error) {
	projects := new([]Project)
	err := c.Request("projects", opt, projects)
	return *projects, err
}

func (c *Client) ListTasks(opt *Filter) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request("tasks", opt, tasks)
	return *tasks, err
}

func (c *Client) GetTask(id int64, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.Request(fmt.Sprintf("tasks/%d", id), opt, task)
	return *task, err
}

// UpdateTask updates a task.
//
// https://asana.com/developers/api-reference/tasks#update
func (c *Client) UpdateTask(id int64, tu TaskUpdate, opt *Filter) (Task, error) {
	task := new(Task)
	err := c.request("PUT", fmt.Sprintf("tasks/%d", id), tu, opt, task)
	return *task, err
}

func (c *Client) ListProjectTasks(projectID int64, opt *Filter) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request(fmt.Sprintf("projects/%d/tasks", projectID), opt, tasks)
	return *tasks, err
}

func (c *Client) ListTaskStories(taskID int64, opt *Filter) ([]Story, error) {
	stories := new([]Story)
	err := c.Request(fmt.Sprintf("tasks/%d/stories", taskID), opt, stories)
	return *stories, err
}

func (c *Client) ListTags(opt *Filter) ([]Tag, error) {
	tags := new([]Tag)
	err := c.Request("tags", opt, tags)
	return *tags, err
}

func (c *Client) GetAuthenticatedUser(opt *Filter) (User, error) {
	user := new(User)
	err := c.Request("users/me", opt, user)
	return *user, err
}

func (c *Client) GetUserByID(id int64, opt *Filter) (User, error) {
	user := new(User)
	err := c.Request(fmt.Sprintf("users/%d", id), opt, user)
	return *user, err
}

func (c *Client) Request(path string, opt *Filter, v interface{}) error {
	return c.request("GET", path, nil, opt, v)
}

// request makes a request to Asana API, using method, at path, sending data with opt filter.
// The response is populated into v, and any error is returned.
func (c *Client) request(method string, path string, data interface{}, opt *Filter, v interface{}) error {
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
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return err
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := &Response{Data: v}
	err = json.NewDecoder(resp.Body).Decode(res)
	if len(res.Errors) > 0 {
		return res.Errors[0]
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
