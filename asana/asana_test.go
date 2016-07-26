package asana

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

var (
	client *Client
	mux    *http.ServeMux
	server *httptest.Server
)

func setup() {
	client = NewClient(nil)
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

func teardown() {
	server.Close()
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil)

	if c.BaseURL.String() != defaultBaseURL {
		t.Errorf("NewClient BaseURL = %v, want %v", c.BaseURL.String(), defaultBaseURL)
	}
	if c.UserAgent != userAgent {
		t.Errorf("NewClient UserAgent = %v, want %v", c.UserAgent, userAgent)
	}
}

func TestNewError(t *testing.T) {
	err := Error{Phrase: "P", Message: "M"}
	if err.Error() != "M - P" {
		t.Errorf("Invalid Error message: %v", err.Error())
	}
}

func TestListWorkspaces(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":1,"name":"Organization 1"},
			{"id":2,"name":"Organization 2"}
		]}`)
	})

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		t.Errorf("ListWorkspaces returned error: %v", err)
	}

	want := []Workspace{
		{ID: 1, Name: "Organization 1"},
		{ID: 2, Name: "Organization 2"},
	}

	if !reflect.DeepEqual(workspaces, want) {
		t.Errorf("ListWorkspaces returned %+v, want %+v", workspaces, want)
	}
}

func TestListUsers(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":1,"email":"test1@asana.com"},
			{"id":2,"email":"test2@asana.com"}
		]}`)
	})

	users, err := client.ListUsers(nil)
	if err != nil {
		t.Errorf("ListUsers returned error: %v", err)
	}

	want := []User{
		{ID: 1, Email: "test1@asana.com"},
		{ID: 2, Email: "test2@asana.com"},
	}

	if !reflect.DeepEqual(users, want) {
		t.Errorf("ListUsers returned %+v, want %+v", users, want)
	}
}

func TestListProjects(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":1,"name":"Project 1"},
			{"id":2,"name":"Project 2"}
		]}`)
	})

	projects, err := client.ListProjects(nil)
	if err != nil {
		t.Errorf("ListProjects returned error: %v", err)
	}

	want := []Project{
		{ID: 1, Name: "Project 1"},
		{ID: 2, Name: "Project 2"},
	}

	if !reflect.DeepEqual(projects, want) {
		t.Errorf("ListProjects returned %+v, want %+v", projects, want)
	}
}

func TestListTasks(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":1,"name":"Task 1"},
			{"id":2,"name":"Task 2"}
		]}`)
	})

	tasks, err := client.ListTasks(nil)
	if err != nil {
		t.Errorf("ListTasks returned error: %v", err)
	}

	want := []Task{
		{ID: 1, Name: "Task 1"},
		{ID: 2, Name: "Task 2"},
	}

	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("ListTasks returned %+v, want %+v", tasks, want)
	}
}

func TestUpdateTask(t *testing.T) {
	setup()
	defer teardown()

	var called int
	defer func() { testCalled(t, called, 1) }()
	mux.HandleFunc("/tasks/1", func(w http.ResponseWriter, r *http.Request) {
		called++
		testMethod(t, r, "PUT")
		testHeader(t, r, "Content-Type", "application/json")
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("error reading request body: %v", err)
		}
		want := `{"data":{"notes":"updated notes"}}`
		if !reflect.DeepEqual(string(b), want) {
			t.Errorf("handler received request body %+v, want %+v", string(b), want)
		}

		fmt.Fprint(w, `{"data":{"id":1,"notes":"updated notes"}}`)
	})

	// TODO: Add this to package API, like go-github, maybe? Think about it first.
	//
	// String is a helper routine that allocates a new string value
	// to store v and returns a pointer to it.
	String := func(v string) *string { return &v }

	task, err := client.UpdateTask(1, TaskUpdate{Notes: String("updated notes")}, nil)
	if err != nil {
		t.Errorf("UpdateTask returned error: %v", err)
	}

	want := Task{ID: 1, Notes: "updated notes"}
	if !reflect.DeepEqual(task, want) {
		t.Errorf("UpdateTask returned %+v, want %+v", task, want)
	}
}

func TestListTags(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
			{"id":1,"name":"Tag 1"},
			{"id":2,"name":"Tag 2"}
		]}`)
	})

	tags, err := client.ListTags(nil)
	if err != nil {
		t.Errorf("ListTags returned error: %v", err)
	}

	want := []Tag{
		{ID: 1, Name: "Tag 1"},
		{ID: 2, Name: "Tag 2"},
	}

	if !reflect.DeepEqual(tags, want) {
		t.Errorf("ListTags returned %+v, want %+v", tags, want)
	}
}

func testMethod(t *testing.T, r *http.Request, want string) {
	if got := r.Method; got != want {
		t.Errorf("Request method: %v, want %v", got, want)
	}
}

func testHeader(t *testing.T, r *http.Request, header string, want string) {
	if got := r.Header.Get(header); got != want {
		t.Errorf("Header.Get(%q) returned %q, want %q", header, got, want)
	}
}

func testCalled(t *testing.T, called int, want int) {
	if got := called; got != want {
		t.Errorf("handler was called %v times, but expected to be called %v times", got, want)
	}
}
