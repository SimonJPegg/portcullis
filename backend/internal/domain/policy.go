package domain

type Policy struct {
	Id      string
	//content string //TODO flesh this out
}

type PolicyResult struct {
	PolicyId string
	Verdict Verdict // TODO: tighten once evaluation logic is implemented — Pending may not belong here
}
