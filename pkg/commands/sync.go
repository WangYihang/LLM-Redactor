package commands

import (
	"io"
	"net/http"
	"os"

	"github.com/wangyihang/llm-prism/pkg/config"
	"github.com/wangyihang/llm-prism/pkg/utils/logging"
)

func Sync(cli *config.CLI, logs *logging.Loggers) {
	logs.System.Info().Str("url", cli.Sync.URL).Msg("syncing rules")
	resp, err := http.Get(cli.Sync.URL)
	if err != nil {
		logs.System.Fatal().Err(err).Msg("failed to download rules")
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.System.Fatal().Err(err).Msg("failed to read response")
	}

	if err := os.WriteFile(cli.RedactorRules, data, 0644); err != nil {
		logs.System.Fatal().Err(err).Msg("failed to save rules")
	}
	logs.System.Info().Str("file", cli.RedactorRules).Msg("sync completed")
}
