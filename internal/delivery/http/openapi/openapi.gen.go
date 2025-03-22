// Package openapi provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package openapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/oapi-codegen/runtime"
	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Defines values for ServerStatus.
const (
	Errored      ServerStatus = "errored"
	Idle         ServerStatus = "idle"
	Initializing ServerStatus = "initializing"
	Running      ServerStatus = "running"
	Starting     ServerStatus = "starting"
	Stopping     ServerStatus = "stopping"
)

// Defines values for ServerConfigDockerType.
const (
	Docker ServerConfigDockerType = "docker"
)

// BaseResource defines model for BaseResource.
type BaseResource struct {
	// Id The unique identifier for the resource
	Id openapi_types.UUID `json:"id"`
}

// NewServer defines model for NewServer.
type NewServer struct {
	Config ServerConfig `json:"config"`
}

// Server defines model for Server.
type Server struct {
	Config ServerConfig `json:"config"`

	// Id The unique identifier for the resource
	Id     openapi_types.UUID `json:"id"`
	Status ServerStatus       `json:"status"`
}

// ServerStatus defines model for Server.Status.
type ServerStatus string

// ServerConfig defines model for ServerConfig.
type ServerConfig struct {
	union json.RawMessage
}

// ServerConfigDocker defines model for ServerConfigDocker.
type ServerConfigDocker struct {
	// Environment The environment variables to set on the server
	Environment []string `json:"environment"`

	// Image The Docker image to use for the server
	Image string `json:"image"`

	// Ports The ports to expose on the server
	Ports []string               `json:"ports"`
	Type  ServerConfigDockerType `json:"type"`

	// Volumes The volumes to mount on the server
	Volumes []string `json:"volumes"`
}

// ServerConfigDockerType defines model for ServerConfigDocker.Type.
type ServerConfigDockerType string

// ServerResponse defines model for ServerResponse.
type ServerResponse struct {
	Server Server `json:"server"`
}

// ServersResponse defines model for ServersResponse.
type ServersResponse struct {
	Servers []Server `json:"servers"`
}

// CreateServerJSONRequestBody defines body for CreateServer for application/json ContentType.
type CreateServerJSONRequestBody = NewServer

// AsServerConfigDocker returns the union data inside the ServerConfig as a ServerConfigDocker
func (t ServerConfig) AsServerConfigDocker() (ServerConfigDocker, error) {
	var body ServerConfigDocker
	err := json.Unmarshal(t.union, &body)
	return body, err
}

// FromServerConfigDocker overwrites any union data inside the ServerConfig as the provided ServerConfigDocker
func (t *ServerConfig) FromServerConfigDocker(v ServerConfigDocker) error {
	b, err := json.Marshal(v)
	t.union = b
	return err
}

// MergeServerConfigDocker performs a merge with any union data inside the ServerConfig, using the provided ServerConfigDocker
func (t *ServerConfig) MergeServerConfigDocker(v ServerConfigDocker) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	merged, err := runtime.JSONMerge(t.union, b)
	t.union = merged
	return err
}

func (t ServerConfig) MarshalJSON() ([]byte, error) {
	b, err := t.union.MarshalJSON()
	return b, err
}

func (t *ServerConfig) UnmarshalJSON(b []byte) error {
	err := t.union.UnmarshalJSON(b)
	return err
}

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// List all servers
	// (GET /api/servers)
	ListServers(w http.ResponseWriter, r *http.Request)
	// Create a new server
	// (POST /api/servers)
	CreateServer(w http.ResponseWriter, r *http.Request)
	// Get a server by ID
	// (GET /api/servers/{id})
	GetServer(w http.ResponseWriter, r *http.Request, id openapi_types.UUID)
}

// Unimplemented server implementation that returns http.StatusNotImplemented for each endpoint.

type Unimplemented struct{}

// List all servers
// (GET /api/servers)
func (_ Unimplemented) ListServers(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Create a new server
// (POST /api/servers)
func (_ Unimplemented) CreateServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Get a server by ID
// (GET /api/servers/{id})
func (_ Unimplemented) GetServer(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
	ErrorHandlerFunc   func(w http.ResponseWriter, r *http.Request, err error)
}

type MiddlewareFunc func(http.Handler) http.Handler

// ListServers operation middleware
func (siw *ServerInterfaceWrapper) ListServers(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.ListServers(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// CreateServer operation middleware
func (siw *ServerInterfaceWrapper) CreateServer(w http.ResponseWriter, r *http.Request) {

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.CreateServer(w, r)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

// GetServer operation middleware
func (siw *ServerInterfaceWrapper) GetServer(w http.ResponseWriter, r *http.Request) {

	var err error

	// ------------- Path parameter "id" -------------
	var id openapi_types.UUID

	err = runtime.BindStyledParameterWithOptions("simple", "id", chi.URLParam(r, "id"), &id, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		siw.ErrorHandlerFunc(w, r, &InvalidParamFormatError{ParamName: "id", Err: err})
		return
	}

	handler := http.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetServer(w, r, id)
	}))

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler.ServeHTTP(w, r)
}

type UnescapedCookieParamError struct {
	ParamName string
	Err       error
}

func (e *UnescapedCookieParamError) Error() string {
	return fmt.Sprintf("error unescaping cookie parameter '%s'", e.ParamName)
}

func (e *UnescapedCookieParamError) Unwrap() error {
	return e.Err
}

type UnmarshalingParamError struct {
	ParamName string
	Err       error
}

func (e *UnmarshalingParamError) Error() string {
	return fmt.Sprintf("Error unmarshaling parameter %s as JSON: %s", e.ParamName, e.Err.Error())
}

func (e *UnmarshalingParamError) Unwrap() error {
	return e.Err
}

type RequiredParamError struct {
	ParamName string
}

func (e *RequiredParamError) Error() string {
	return fmt.Sprintf("Query argument %s is required, but not found", e.ParamName)
}

type RequiredHeaderError struct {
	ParamName string
	Err       error
}

func (e *RequiredHeaderError) Error() string {
	return fmt.Sprintf("Header parameter %s is required, but not found", e.ParamName)
}

func (e *RequiredHeaderError) Unwrap() error {
	return e.Err
}

type InvalidParamFormatError struct {
	ParamName string
	Err       error
}

func (e *InvalidParamFormatError) Error() string {
	return fmt.Sprintf("Invalid format for parameter %s: %s", e.ParamName, e.Err.Error())
}

func (e *InvalidParamFormatError) Unwrap() error {
	return e.Err
}

type TooManyValuesForParamError struct {
	ParamName string
	Count     int
}

func (e *TooManyValuesForParamError) Error() string {
	return fmt.Sprintf("Expected one value for %s, got %d", e.ParamName, e.Count)
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL          string
	BaseRouter       chi.Router
	Middlewares      []MiddlewareFunc
	ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	if options.ErrorHandlerFunc == nil {
		options.ErrorHandlerFunc = func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
		ErrorHandlerFunc:   options.ErrorHandlerFunc,
	}

	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/servers", wrapper.ListServers)
	})
	r.Group(func(r chi.Router) {
		r.Post(options.BaseURL+"/api/servers", wrapper.CreateServer)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/servers/{id}", wrapper.GetServer)
	})

	return r
}

type ListServersRequestObject struct {
}

type ListServersResponseObject interface {
	VisitListServersResponse(w http.ResponseWriter) error
}

type ListServers200JSONResponse ServersResponse

func (response ListServers200JSONResponse) VisitListServersResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type ListServers500Response struct {
}

func (response ListServers500Response) VisitListServersResponse(w http.ResponseWriter) error {
	w.WriteHeader(500)
	return nil
}

type CreateServerRequestObject struct {
	Body *CreateServerJSONRequestBody
}

type CreateServerResponseObject interface {
	VisitCreateServerResponse(w http.ResponseWriter) error
}

type CreateServer201JSONResponse ServerResponse

func (response CreateServer201JSONResponse) VisitCreateServerResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)

	return json.NewEncoder(w).Encode(response)
}

type CreateServer400Response struct {
}

func (response CreateServer400Response) VisitCreateServerResponse(w http.ResponseWriter) error {
	w.WriteHeader(400)
	return nil
}

type CreateServer500Response struct {
}

func (response CreateServer500Response) VisitCreateServerResponse(w http.ResponseWriter) error {
	w.WriteHeader(500)
	return nil
}

type GetServerRequestObject struct {
	Id openapi_types.UUID `json:"id"`
}

type GetServerResponseObject interface {
	VisitGetServerResponse(w http.ResponseWriter) error
}

type GetServer200JSONResponse ServerResponse

func (response GetServer200JSONResponse) VisitGetServerResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	return json.NewEncoder(w).Encode(response)
}

type GetServer404Response struct {
}

func (response GetServer404Response) VisitGetServerResponse(w http.ResponseWriter) error {
	w.WriteHeader(404)
	return nil
}

type GetServer500Response struct {
}

func (response GetServer500Response) VisitGetServerResponse(w http.ResponseWriter) error {
	w.WriteHeader(500)
	return nil
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// List all servers
	// (GET /api/servers)
	ListServers(ctx context.Context, request ListServersRequestObject) (ListServersResponseObject, error)
	// Create a new server
	// (POST /api/servers)
	CreateServer(ctx context.Context, request CreateServerRequestObject) (CreateServerResponseObject, error)
	// Get a server by ID
	// (GET /api/servers/{id})
	GetServer(ctx context.Context, request GetServerRequestObject) (GetServerResponseObject, error)
}

type StrictHandlerFunc = strictnethttp.StrictHTTPHandlerFunc
type StrictMiddlewareFunc = strictnethttp.StrictHTTPMiddlewareFunc

type StrictHTTPServerOptions struct {
	RequestErrorHandlerFunc  func(w http.ResponseWriter, r *http.Request, err error)
	ResponseErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)
}

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares, options: StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		},
	}}
}

func NewStrictHandlerWithOptions(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc, options StrictHTTPServerOptions) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares, options: options}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
	options     StrictHTTPServerOptions
}

// ListServers operation middleware
func (sh *strictHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	var request ListServersRequestObject

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.ListServers(ctx, request.(ListServersRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "ListServers")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(ListServersResponseObject); ok {
		if err := validResponse.VisitListServersResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// CreateServer operation middleware
func (sh *strictHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	var request CreateServerRequestObject

	var body CreateServerJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sh.options.RequestErrorHandlerFunc(w, r, fmt.Errorf("can't decode JSON body: %w", err))
		return
	}
	request.Body = &body

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.CreateServer(ctx, request.(CreateServerRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "CreateServer")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(CreateServerResponseObject); ok {
		if err := validResponse.VisitCreateServerResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// GetServer operation middleware
func (sh *strictHandler) GetServer(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var request GetServerRequestObject

	request.Id = id

	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, request interface{}) (interface{}, error) {
		return sh.ssi.GetServer(ctx, request.(GetServerRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "GetServer")
	}

	response, err := handler(r.Context(), w, r, request)

	if err != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, err)
	} else if validResponse, ok := response.(GetServerResponseObject); ok {
		if err := validResponse.VisitGetServerResponse(w); err != nil {
			sh.options.ResponseErrorHandlerFunc(w, r, err)
		}
	} else if response != nil {
		sh.options.ResponseErrorHandlerFunc(w, r, fmt.Errorf("unexpected response type: %T", response))
	}
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/7RWW2vjRhT+K8NpoS/CchsXgmAfdpNlMSzJkg19CaZMRsf2bKWZ2bnYcYP/ezkzknyR",
	"HCeQvpi5nDmX7zvfkZ9B6Npohco7KJ7BiSXWPC4/cYd36HSwAmlvrDZovcR4K0v6LdEJK42XWkEB90tk",
	"QcmfAZksUXk5l2jZXFvml8hs6yuDubY191BACLKEDPzGIBTgvJVqAdttBhZ/BmmxhOKBQs06G/34A4WH",
	"bQY3uP6OdoW2n5zQai4XtPrV4hwK+CXflZk3Nebp9VWyPQ7auBgKvIvKq+p2DsXDy3F2iW6zly0PICfj",
	"w7Kc5z7EFapQR2iU9JJX8l/CLQNZVgSv89z6dGKDUmnlvDYmLdFaTWXOziHfBOyDcBqWqw56rfAV4Oy/",
	"utbiH0JpduStOe/RjGolrVY1Kj/cjHsGbMWt5I8VOuY1c+iZVrEtXaImA3zitamQ6v52e3f/4XJ8OYYM",
	"bm6vP//9+eavD8bqMojofZaB9FjHJI4Q7IDh1vIN7WXNFzicXyqMRQtKKzjs5NLl1QtgtE1i7TuMV+QJ",
	"n4x2+FKNl+OCKsy9MJDBZHJRXE4mF3H7pvLSfteSZSJrNpD4SlehxhOpN5eUfK2DepGffMVtvl6v86Wv",
	"q+JgBxnk6EWuFlI9pd8RabkYPH1LqUfaiJctu7vaWnqyg+48rZc7dEYrNzBhXTdmzuunr9x0fDquOxc4",
	"TfkWm9ekcAaw1m0/J7KUaq6HGkM6Jl1shI/fpqzUIhCgnO47qaQEjA5iSVYjNvW/ORJTSd20QIWWe+yc",
	"iErSQIj3j5ueh6uv0xHJTnpqNjhyTlSjdSm98Wg8GlPh2qDiRkIBF/EoA8P9MiKXcyPzPUgXGIcVAR6r",
	"mJZQwFfpfMMLDe2Gmmj/x3jcfNJ8M+e4MZUU8XH+w1Ei7Xf7dUztqI/I96XYZMvWaGkeBVVSjX+mRA7N",
	"PyomlUereNU8Y/HrwrQQwRLzFMKFuuZ20xTKeNUauzTP3AAkVxa5x++t+qmT0PlPuty8Gxx7H+bDZvU2",
	"4LbHw+/vzMPraGBr7piIYJTMBSHQuXmoqjh8J0Oc3Mc/WxGu+FiqFa/k+3GYmGGcKVy345ks9js9f5bl",
	"9mS7f0HfEWu45TX6qI6HZ5CUEYkHMlC8JgXGv4iH5GR7QJ/7Mzn73wX1BiI7NU3Gk2Hm9oyV9u8svy/o",
	"GW8tHzdsek05b/8LAAD//26GyeMGDAAA",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
