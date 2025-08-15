package api

import "time"

type StartJobRequest struct {
	Port        string                 `json:"port"`
	Direction   string                 `json:"direction"` // ingress|egress|both
	SpanMethod  string                 `json:"span_method"` // span|erspan
	VLAN        *int                   `json:"vlan,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	SampleRate  int                    `json:"sample_rate"`
	DurationSec int                    `json:"duration_sec"`
	OTLPExport  bool                   `json:"otlp_export"`
	ResultDetail string                `json:"result_detail"` // summary|flows|pcaplike
}

type StartJobResponse struct {
	JobID    string `json:"job_id"`
	Status   string `json:"status"`
	Interface string `json:"interface"`
}

type JobStatus struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Port      string    `json:"port"`
	Interface string    `json:"interface"`
}

type StopJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type JobResults struct {
	WindowSec int            `json:"window_sec"`
	Packets   uint64         `json:"packets_total"`
	Bytes     uint64         `json:"bytes_total"`
	Errors    map[string]uint64 `json:"errors"`
	TopFlows  []TopFlow      `json:"top_flows"`
	LatencyHistogramNs Histogram `json:"latency_histogram_ns"`
	OTLPExport OTLPInfo      `json:"otel_export"`
}

type TopFlow struct {
	FiveTuple string `json:"5tuple"`
	Pkts      uint64 `json:"pkts"`
	Bytes     uint64 `json:"bytes"`
}

type Histogram struct {
	Bounds []uint64 `json:"bounds"`
	Counts []uint64 `json:"counts"`
}

type OTLPInfo struct {
	Exported bool   `json:"exported"`
	Endpoint string `json:"endpoint"`
}
