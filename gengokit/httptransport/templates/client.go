package templates

// ClientEncodeTemplate is the template for generating the client-side encoding
// function for a particular Binding.
var ClientEncodeTemplate = `
{{- with $binding := . -}}
	// EncodeHTTP{{$binding.Label}}Request is a transport/http.EncodeRequestFunc
	// that encodes a {{ToLower $binding.Parent.Name}} request into the various portions of
	// the http request (path, query, and body).
	func EncodeHTTP{{$binding.Label}}Request(_ context.Context, r *http.Request, request interface{}) error {
		fmt.Printf("Encoding request %v\n", request)
		strval := ""
		_ = strval
		req := request.(*pb.{{GoName $binding.Parent.RequestType}})
		_ = req

		// Set the path parameters
		path := strings.Join([]string{
		{{- range $section := $binding.PathSections}}
			{{$section}},
		{{- end}}
		}, "/")
		u, err := url.Parse(path)
		if err != nil {
			return errors.Wrapf(err, "couldn't unmarshal path %q", path)
		}
		r.URL.RawPath = u.RawPath
		r.URL.Path = u.Path

		// Set the query parameters
		values := r.URL.Query()
		var tmp []byte
		_ = tmp
		{{- range $field := $binding.Fields }}
			{{- if eq $field.Location "query"}}
				{{if or (not $field.IsBaseType) $field.Repeated}}
					tmp, err = json.Marshal(req.{{$field.CamelName}})
					if err != nil {
						return errors.Wrap(err, "failed to marshal req.{{$field.CamelName}}")
					}
					strval = string(tmp)
					values.Add("{{$field.Name}}", strval)
				{{else}}
					values.Add("{{$field.Name}}", fmt.Sprint(req.{{$field.CamelName}}))
				{{- end }}
			{{- end }}
		{{- end}}

		r.URL.RawQuery = values.Encode()

		// Set the body parameters
		var buf bytes.Buffer
		toRet := request.(*pb.{{GoName $binding.Parent.RequestType}})
		{{- range $field := $binding.Fields -}}
			{{if eq $field.Location "body"}}
				{{/* Only set the fields which should be in the body, so all
				others will be omitted due to emptiness */}}
				toRet.{{$field.CamelName}} = req.{{$field.CamelName}}
			{{end}}
		{{- end }}
		if err := json.NewEncoder(&buf).Encode(toRet); err != nil {
			return errors.Wrapf(err, "couldn't encode body as json %v", toRet)
		}
		r.Body = ioutil.NopCloser(&buf)
		fmt.Printf("URL: %v\n", r.URL)
		return nil
	}
{{- end -}}
`

var ClientTemplate = `
// Code generated by truss.
// Rerunning truss will overwrite this file.
// DO NOT EDIT!

// Package http provides an HTTP client for the {{.Service.Name}} service.
package http

import (
	"net/url"
	"strings"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	// This Service
	"{{.ImportPath -}} /svc"
	pb "{{.PBImportPath -}}"
)

var (
	_ = endpoint.Chain
	_ = httptransport.NewClient
)

// New returns a service backed by an HTTP server living at the remote
// instance. We expect instance to come from a service discovery system, so
// likely of the form "host:port".
func New(instance string, options ...ClientOption) (pb.{{GoName .Service.Name}}Server, error) {
	var cc clientConfig

	for _, f := range options {
		err := f(&cc)
		if err != nil {
			return nil, errors.Wrap(err, "cannot apply option") }
	}

	{{ if .HTTPHelper.Methods }}
		clientOptions := []httptransport.ClientOption{
			httptransport.ClientBefore(
				contextValuesToHttpHeaders(cc.headers)),
		}
	{{ end }}

	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}
	_ = u

	{{range $method := .HTTPHelper.Methods}}
		{{range $binding := $method.Bindings}}
			var {{$binding.Label}}Endpoint endpoint.Endpoint
			{
				{{$binding.Label}}Endpoint = httptransport.NewClient(
					"{{$binding.Verb}}",
					copyURL(u, "{{$binding.BasePath}}"),
					svc.EncodeHTTP{{$binding.Label}}Request,
					svc.DecodeHTTP{{$method.Name}}Response,
					clientOptions...,
				).Endpoint()
			}
		{{- end}}
	{{- end}}

	return svc.Endpoints{
	{{range $method := .HTTPHelper.Methods -}}
		{{range $binding := $method.Bindings -}}
			{{$method.Name}}Endpoint:    {{$binding.Label}}Endpoint,
		{{end}}
	{{- end}}
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

type clientConfig struct {
	headers []string
}

// ClientOption is a function that modifies the client config
type ClientOption func(*clientConfig) error

// CtxValuesToSend configures the http client to pull the specified keys out of
// the context and add them to the http request as headers.  Note that keys
// will have net/http.CanonicalHeaderKey called on them before being send over
// the wire and that is the form they will be available in the server context.
func CtxValuesToSend(keys ...string) ClientOption {
	return func(o *clientConfig) error {
		o.headers = keys
		return nil
	}
}

func contextValuesToHttpHeaders(keys []string) httptransport.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		for _, k := range keys {
			if v, ok := ctx.Value(k).(string); ok {
				r.Header.Set(k, v)
			}
		}

		return ctx
	}
}
`