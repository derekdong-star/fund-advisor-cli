package llm

import "fmt"

func validateCandidateRerankResponse(req CandidateRerankRequest, resp *CandidateRerankResponse) error {
	if resp == nil {
		return fmt.Errorf("candidate rerank response is nil")
	}
	if len(resp.Rankings) != len(req.Candidates) {
		return fmt.Errorf("candidate rerank response count = %d, want %d", len(resp.Rankings), len(req.Candidates))
	}
	allowed := make(map[string]struct{}, len(req.Candidates))
	for _, item := range req.Candidates {
		allowed[item.FundCode] = struct{}{}
	}
	seenCodes := make(map[string]struct{}, len(resp.Rankings))
	seenRanks := make(map[int]struct{}, len(resp.Rankings))
	for _, ranking := range resp.Rankings {
		if ranking.Rank <= 0 {
			return fmt.Errorf("candidate %s has invalid rank %d", ranking.FundCode, ranking.Rank)
		}
		if _, exists := seenRanks[ranking.Rank]; exists {
			return fmt.Errorf("rank %d appears multiple times in rerank response", ranking.Rank)
		}
		seenRanks[ranking.Rank] = struct{}{}
		if _, ok := allowed[ranking.FundCode]; !ok {
			return fmt.Errorf("candidate %s is not in the rerank request", ranking.FundCode)
		}
		if _, exists := seenCodes[ranking.FundCode]; exists {
			return fmt.Errorf("candidate %s appears multiple times in rerank response", ranking.FundCode)
		}
		seenCodes[ranking.FundCode] = struct{}{}
	}
	for rank := 1; rank <= len(req.Candidates); rank++ {
		if _, ok := seenRanks[rank]; !ok {
			return fmt.Errorf("rank %d is missing from rerank response", rank)
		}
	}
	return nil
}

func ValidateCandidateRerankResponseForCLI(req CandidateRerankRequest, resp *CandidateRerankResponse) error {
	return validateCandidateRerankResponse(req, resp)
}
