package openevse

type CommandResult struct {
	CMD string `json:"cmd"`
	RET string `json:"ret"`
}

type StatusResult struct {
	Amp      int64  `json:"amp"`
	Watthour int64  `json:"watthour"`
	Wattsec  int64  `json:"wattsec"`
	Status   string `json:"status"`
}

type OverrideResult struct {
	Msg string `json:"msg"`
}
