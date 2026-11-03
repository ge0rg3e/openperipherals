package sysstats

// GPU is one GPU's load.
type GPU struct {
	Name      string `json:"name"`
	Usage     int    `json:"usage"` // %
	VRAMUsed  uint64 `json:"vramUsed"`
	VRAMTotal uint64 `json:"vramTotal"`
}

// Stats is one sample.
type Stats struct {
	CPU      int    `json:"cpu"` // %, -1 on the first sample
	RAMUsed  uint64 `json:"ramUsed"`
	RAMTotal uint64 `json:"ramTotal"`
	GPUs     []GPU  `json:"gpus"`
}
