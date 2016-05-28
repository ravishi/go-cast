package ctrl

type PayloadHeaders struct {
	Type      string `json:"type"`
	RequestId *int   `json:"requestId,omitempty"`
}
