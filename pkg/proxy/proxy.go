package proxy

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/wangyihang/llm-prism/pkg/utils/ctxkeys"
)

// ContentRedactor defines the redaction capabilities required by the proxy layer.
// This interface decouples the proxy from the concrete redactor implementation.
type ContentRedactor interface {
	RedactRequest(ctx context.Context, body []byte) ([]byte, error)
	WrapSSEReader(ctx context.Context, rc io.ReadCloser) io.ReadCloser
}

// New creates a new goproxy.ProxyHttpServer configured for LLM traffic interception.
func New(rdr ContentRedactor, sysLog, trafficLog zerolog.Logger, sessionDir string) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = false

	caPath, err := GenerateAndSetCA(sessionDir)
	if err != nil {
		sysLog.Warn().Err(err).Msg("failed to generate session CA certificate")
	} else {
		sysLog.Info().Str("path", caPath).Msg("session CA certificate generated and applied")
	}

	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	proxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		requestID := uuid.New().String()
		ctx.UserData = map[string]interface{}{
			"request_id": requestID,
			"start_time": time.Now(),
		}

		var requestBody []byte
		if r.Body != nil {
			var err error
			requestBody, err = io.ReadAll(r.Body)
			if err == nil {
				reqCtx := context.Background()
				reqCtx = context.WithValue(reqCtx, ctxkeys.RequestID, requestID)
				reqCtx = context.WithValue(reqCtx, ctxkeys.Source, "request")
				reqCtx = context.WithValue(reqCtx, ctxkeys.Host, r.Host)
				reqCtx = context.WithValue(reqCtx, ctxkeys.Path, r.URL.Path)
				reqCtx = context.WithValue(reqCtx, ctxkeys.Method, r.Method)

				// Skip redaction if rdr is nil
				if rdr != nil {
					redacted, err := rdr.RedactRequest(reqCtx, requestBody)
					if err == nil {
						r.Body = io.NopCloser(bytes.NewReader(redacted))
						r.ContentLength = int64(len(redacted))
						requestBody = redacted
					} else {
						r.Body = io.NopCloser(bytes.NewReader(requestBody))
					}
				} else {
					r.Body = io.NopCloser(bytes.NewReader(requestBody))
				}
			}
		}

		ctx.UserData.(map[string]interface{})["request_body"] = requestBody

		return r, nil
	})

	proxy.OnResponse().DoFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		userData := ctx.UserData.(map[string]interface{})
		requestID := userData["request_id"].(string)
		startTime := userData["start_time"].(time.Time)
		requestBody := userData["request_body"].([]byte)

		var responseBody []byte
		if resp != nil && resp.Body != nil {
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "text/event-stream") && rdr != nil {
				resCtx := context.Background()
				resCtx = context.WithValue(resCtx, ctxkeys.RequestID, requestID)
				resCtx = context.WithValue(resCtx, ctxkeys.Source, "response_sse")

				resp.Body = rdr.WrapSSEReader(resCtx, resp.Body)
			} else {
				var err error
				responseBody, err = io.ReadAll(resp.Body)
				if err == nil {
					resp.Body = io.NopCloser(bytes.NewReader(responseBody))
				}
			}
		}

		reqEvt := zerolog.Dict().Str("id", requestID).Str("method", ctx.Req.Method).Str("path", ctx.Req.URL.Path).Str("host", ctx.Req.Host)
		EnrichLogEvent(reqEvt, requestBody, ctx.Req.Header, sysLog)

		resEvt := zerolog.Dict().Int("status", resp.StatusCode)
		EnrichLogEvent(resEvt, responseBody, resp.Header, sysLog)

		trafficLog.Info().
			Str("id", requestID).
			Dur("duration", time.Since(startTime)).
			Dict("http", zerolog.Dict().Dict("request", reqEvt).Dict("response", resEvt)).
			Msg("")

		return resp
	})

	return proxy
}
