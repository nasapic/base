package base

import (
	"fmt"
)

type (
	Endpoint struct {
		Worker
		name    string
		service Service
	}
)

type (
	ContextKey string
)

type (
	Identifiable interface {
		GetSlug() string
	}
)

const (
	SessionCtxKey ContextKey = "session"
)

const (
	GetMethod    = "GET"
	PostMethod   = "POST"
	PutMethod    = "PUT"
	PatchMethod  = "PATCH"
	DeleteMethod = "DELETE"
)

const (
	sessionCookieKey = "session"
)

const (
	SessionKey = "session"
)

func NewEndpoint(name string, log Logger) *Endpoint {
	return &Endpoint{
		Worker: NewWorker(name, log),
		name:   name,
	}
}

func (ep *Endpoint) Name() string {
	return ep.name
}

func (ep *Endpoint) SetName(name string) {
	ep.name = name
}

func (ep *Endpoint) Service() Service {
	return ep.service
}

func (ep Endpoint) SetService(s Service) {
	ep.service = s
}

// Resource path functions

// IndexPath returns index path under resource root path.
func IndexPath() string {
	return ""
}

// EditPath returns edit path under resource root path.
func EditPath() string {
	return "/{id}/edit"
}

// NewPath returns new path under resource root path.
func NewPath() string {
	return "/new"
}

// ShowPath returns show path under resource root path.
func ShowPath() string {
	return "/{id}"
}

// CreatePath returns create path under resource root path.
func CreatePath() string {
	return ""
}

// UpdatePath returns update path under resource root path.
func UpdatePath() string {
	return "/{id}"
}

// DeletePath returns delete path under resource root path.
func DeletePath() string {
	return "/{id}"
}

// ResPath
func ResPath(rootPath string) string {
	return "/" + rootPath + IndexPath()
}

// ResPathEdit
func ResPathEdit(rootPath string, r Identifiable) string {
	return fmt.Sprintf("/%s/%s/edit", rootPath, r.GetSlug())
}

// ResPathNew
func ResPathNew(rootPath string) string {
	return fmt.Sprintf("/%s/new", rootPath)
}

// ResPathInitDelete
func ResPathInitDelete(rootPath string, r Identifiable) string {
	return fmt.Sprintf("/%s/%s/init-delete", rootPath, r.GetSlug())
}

// ResPathSlug
func ResPathSlug(rootPath string, r Identifiable) string {
	return fmt.Sprintf("/%s/%s", rootPath, r.GetSlug())
}

// Admin
func ResAdmin(path, adminPathPfx string) string {
	return fmt.Sprintf("/%s/%s", adminPathPfx, path)
}
