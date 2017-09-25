package oas2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// MiddlewareFn describes middleware function.
type MiddlewareFn func(next http.Handler) http.Handler

// Middleware describes a middleware that can be applied to a http.handler.
type Middleware interface {
	Apply(next http.Handler) http.Handler
}

// TODO: don't use raw errHandler, make validator less complex
// NewQueryValidator returns new Middleware that validates request query
// parameters against OpenAPI 2.0 spec.
func NewQueryValidator(errHandler func(w http.ResponseWriter, errs []error)) Middleware {
	return queryValidatorMiddleware{
		errHandler:      errHandler,
		continueOnError: false, // TODO: make controllable
	}
}

type queryValidatorMiddleware struct {
	errHandler      func(w http.ResponseWriter, errs []error)
	continueOnError bool
}

func (m queryValidatorMiddleware) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		op := GetOperation(req)
		if op == nil {
			next.ServeHTTP(w, req)
			return
		}

		if errs := ValidateQuery(op.Parameters, req.URL.Query()); len(errs) > 0 {
			m.errHandler(w, errs)
			if !m.continueOnError {
				return
			}
		}

		next.ServeHTTP(w, req)
	})
}

// NewBodyValidator returns new Middleware that validates request body
// against parameters defined in OpenAPI 2.0 spec.
func NewBodyValidator(errHandler func(w http.ResponseWriter, errs []error)) Middleware {
	return bodyValidatorMiddleware{
		errHandler: errHandler,
	}
}

type bodyValidatorMiddleware struct {
	errHandler func(w http.ResponseWriter, errs []error)
}

func (m bodyValidatorMiddleware) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Body == http.NoBody {
			next.ServeHTTP(w, req)
			return
		}

		op := GetOperation(req)
		if op == nil {
			next.ServeHTTP(w, req)
			return
		}

		// Read req.Body using io.TeeReader, so it can be read again
		// in the actual request handler.

		var b bytes.Buffer
		tr := io.TeeReader(req.Body, &b)
		defer req.Body.Close()

		var body interface{}
		if err := json.NewDecoder(tr).Decode(&body); err != nil {
			m.errHandler(w, []error{fmt.Errorf("Body contains invalid json")})
			return
		}

		// Validate body
		if errs := ValidateBody(op.Parameters, body); len(errs) > 0 {
			m.errHandler(w, errs)
			return
		}

		// Replace the body so it can be read again.
		req.Body = ioutil.NopCloser(&b)

		next.ServeHTTP(w, req)
	})
}

// NewPathParameterExtractor returns new Middleware that extracts parameters
// defined in OpenAPI 2.0 spec as path parameters from path.
func NewPathParameterExtractor(extractor func(r *http.Request, key string) string) Middleware {
	return pathParameterExtractor{extractor}
}

type pathParameterExtractor struct {
	extractor func(r *http.Request, key string) string
}

func (m pathParameterExtractor) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		op := GetOperation(req)
		if op == nil {
			next.ServeHTTP(w, req)
			return
		}

		for _, p := range op.Parameters {
			if p.In != "path" {
				continue
			}

			value, err := ConvertPrimitive(m.extractor(req, p.Name), p.Type, p.Format)
			if err == nil {
				req = req.WithContext(
					context.WithValue(req.Context(), contextKeyPathParam(p.Name), value),
				)
			}
		}

		next.ServeHTTP(w, req)
	})
}

// GetPathParam returns a path parameter by name from a request.
// For example, a handler defined on a path "/pet/{id}" gets a request with
// path "/pet/12" - in this case GetPathParam(req, "id") returns 12.
func GetPathParam(req *http.Request, name string) interface{} {
	return req.Context().Value(contextKeyPathParam(name))
}

type contextKeyPathParam string

// NewResponseBodyValidator returns new Middleware that validates response body
// against schema defined in OpenAPI 2.0 spec.
func NewResponseBodyValidator(errHandler func(w http.ResponseWriter, errs []error)) Middleware {
	return responseBodyValidator{errHandler}
}

type responseBodyValidator struct {
	errHandler func(w http.ResponseWriter, errs []error)
}

func (m responseBodyValidator) Apply(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		op := GetOperation(req)
		if op == nil {
			next.ServeHTTP(w, req)
			return
		}

		rr := NewResponseRecorder(w)

		next.ServeHTTP(rr, req)

		responseSpec, ok := op.Responses.StatusCodeResponses[rr.Status()]
		if !ok {
			// TODO: should notify package user that there is no response spec.
			return
		}

		if responseSpec.Schema == nil {
			// It may be ok for responses like 204.
			return
		}

		var body interface{}
		if err := json.Unmarshal(rr.Payload(), &body); err != nil {
			// TODO: should notify package user about the error.
			return
		}

		if errs := ValidateBySchema(responseSpec.Schema, body); len(errs) > 0 {
			m.errHandler(w, errs)
		}
	})
}
