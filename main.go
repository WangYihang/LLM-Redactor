package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/wangyihang/llm-prism/pkg/llms/providers"
	"github.com/wangyihang/llm-prism/pkg/utils/logging"
	"github.com/wangyihang/llm-prism/pkg/utils/version"
)

type CLI struct {
	LogFile string `help:"The log file path." env:"LLM_PRISM_LOG_FILE" default:"llm-prism.jsonl"`
	Run     struct {
		ApiURL string `help:"The API base URL." env:"LLM_PRISM_API_URL" default:"https://api.deepseek.com/anthropic"`
		ApiKey string `help:"The API key." env:"LLM_PRISM_API_KEY" required:""`
		Host   string `help:"The host to listen on." env:"LLM_PRISM_HOST" default:"0.0.0.0"`
		Port   int    `help:"The port to listen on." env:"LLM_PRISM_PORT" default:"4000"`
	} `cmd:"" help:"Run the proxy server."`
	Version struct {
	} `cmd:"" help:"Print version information."`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("llm-prism"),
		kong.Description("A proxy server for LLM API requests."),
		kong.UsageOnError(),
	)

	logger := logging.GetLogger(cli.LogFile)

	switch ctx.Command() {
	case "run":
		logger.Info().
			Str("api_url", cli.Run.ApiURL).
			Str("host", cli.Run.Host).
			Int("port", cli.Run.Port).
			Str("log_file", cli.LogFile).
			Msg("starting proxy server")
		runProxy(logger, cli.Run.ApiURL, cli.Run.ApiKey, cli.Run.Host, cli.Run.Port)
	case "version":
		logger.Info().
			Str("version", version.GetVersionInfo().Version).
			Str("commit", version.GetVersionInfo().Commit).
			Str("date", version.GetVersionInfo().Date).
			Msg("version information")
		fmt.Println(version.GetVersionInfo().JSON())
	}
}

type responseCapture struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseCapture) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseCapture) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func decodeBody(data []byte, header http.Header) string {

	contentEncoding := header.Get("Content-Encoding")

	if strings.Contains(contentEncoding, "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {

			return fmt.Sprintf("<gzip_decode_error: %v>", err)
		}
		defer func() {
			if err := reader.Close(); err != nil {
				slog.Error("failed to close gzip reader", "error", err)
			}
		}()
		decoded, err := io.ReadAll(reader)
		if err != nil {
			return fmt.Sprintf("<gzip_read_error: %v>", err)
		}
		return string(decoded)
	}

	return string(data)
}

func runProxy(logger zerolog.Logger, apiURL, apiKey, host string, port int) {
	baseURL, err := url.Parse(apiURL)
	if err != nil {
		logger.Error().Err(err).Msg("failed to parse API URL")
		return
	}
	deepseekProvider := providers.NewDeepseekProvider(baseURL, apiKey)
	proxy := httputil.NewSingleHostReverseProxy(deepseekProvider.BaseURL)
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		deepseekProvider.Director(req)
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Info().Str("address", addr).Msg("proxy server started")

	err = http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		reqBuf := new(bytes.Buffer)
		r.Body = io.NopCloser(io.TeeReader(r.Body, reqBuf))

		resCapture := &responseCapture{
			ResponseWriter: w,
			body:           new(bytes.Buffer),
			statusCode:     200,
		}

		proxy.ServeHTTP(resCapture, r)

		respHeader := resCapture.Header()

		respBodyStr := decodeBody(resCapture.body.Bytes(), respHeader)

		reqBodyStr := decodeBody(reqBuf.Bytes(), r.Header)

		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", resCapture.statusCode).
			Dur("duration", time.Since(start)).
			Str("request_body", reqBodyStr).
			Str("response_body", respBodyStr).
			Msg("http interaction")
	}))

	if err != nil {
		logger.Error().Err(err).Msg("failed to start proxy server")
		return
	}
}
