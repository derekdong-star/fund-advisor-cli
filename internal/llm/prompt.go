package llm

import (
	"fmt"
	"strings"
)

func CandidateRerankSystemPrompt() string {
	return strings.Join([]string{
		"You are a portfolio candidate reranker.",
		"Only rerank the provided candidates.",
		"Do not add or remove candidates.",
		"Return valid JSON only.",
		"The JSON schema is: {\"rankings\":[{\"fund_code\":string,\"rank\":int,\"score\":number,\"reason\":string}]}",
		"Use one concise sentence per reason.",
	}, " ")
}

func BuildCandidateRerankPrompt(req CandidateRerankRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Portfolio: %s\n", req.PortfolioName)
	fmt.Fprintf(&b, "Run Date: %s\n", req.RunDate.Format("2006-01-02"))
	b.WriteString("Task: rerank the existing candidate replacements only. Keep the rule engine constraints unchanged.\n")
	b.WriteString("Return JSON only with the rankings array.\n")
	for _, item := range req.Candidates {
		fmt.Fprintf(&b, "- %s (%s): score=%d, 20D=%.2f%%, 60D=%.2f%%, 120D=%.2f%%, category=%s, role=%s, benchmark=%s, replace_for=%s, rule_reason=%s\n",
			item.FundName,
			item.FundCode,
			item.Score,
			item.Return20D*100,
			item.Return60D*100,
			item.Return120D*100,
			item.Category,
			item.Role,
			item.Benchmark,
			strings.Join(item.ReplaceFor, "/"),
			item.RuleReason,
		)
	}
	return b.String()
}
