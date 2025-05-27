package internal

// RechartsData represents data for a single column in a Recharts chart.
type RechartsData struct {
	XAxis  string         `json:"xAxis" yaml:"xAxis"`
	Values map[string]int `json:"values" yaml:"values"`
}

// RechartsColumn represents data for a single column in a Recharts chart.
type RechartsColumn struct {
	Column []RechartsData `json:"column" yaml:"column"`
}

// RechartsChart represents a Recharts chart.
type RechartsChart struct {
	Chart []RechartsColumn `json:"chart" yaml:"chart"`
}

// RechartsResponse represents a response for a Recharts chart.
type RechartsResponse struct {
	Charts []RechartsChart `json:"charts" yaml:"charts"`
}
