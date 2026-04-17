package docs

import (
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type ReportArtifact struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type PublishState struct {
	LastRunDate string                    `json:"last_run_date"`
	Latest      map[string]ReportArtifact `json:"latest,omitempty"`
	Archive     map[string]ReportArtifact `json:"archive,omitempty"`
}

type ExportResult struct {
	GeneratedAt time.Time                 `json:"generated_at"`
	DocsRoot    string                    `json:"docs_root"`
	Latest      map[string]ReportArtifact `json:"latest,omitempty"`
	Archive     map[string]ReportArtifact `json:"archive,omitempty"`
}

type PublishInput struct {
	Analysis      *model.AnalysisReport
	Plan          *model.DCAPlanReport
	Backtest      *model.BacktestReport
	BacktestError string
}
