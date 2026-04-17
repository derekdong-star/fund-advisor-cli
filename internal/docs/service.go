package docs

import "github.com/derekdong-star/fund-advisor-cli/internal/config"

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Export(input PublishInput) (*ExportResult, error) {
	return exportReports(s.cfg, input)
}

func (s *Service) BuildIndex(input PublishInput, result *ExportResult) error {
	return buildIndex(s.cfg, input, result)
}

func (s *Service) Validate() error {
	return validateTree(s.cfg, docsRoot(s.cfg))
}

func (s *Service) Publish(input PublishInput) (*ExportResult, error) {
	result, err := s.Export(input)
	if err != nil {
		return nil, err
	}
	if input.Analysis != nil {
		if err := pruneArchivedReports(docsRoot(s.cfg), input.Analysis.Summary.RunDate, s.cfg.Publishing.GitBook.RetainDays); err != nil {
			return nil, err
		}
	}
	if err := s.BuildIndex(input, result); err != nil {
		return nil, err
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return result, nil
}
