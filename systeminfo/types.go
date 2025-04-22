package systeminfo

type Dogu struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Volume  volume `json:"volume"`
}

type volume struct {
	SizeInBytes int64 `json:"sizeInBytes"`
}

type component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type SystemInfo struct {
	Dogus      []Dogu      `json:"dogus"`
	Components []component `json:"components"`
}
