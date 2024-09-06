package segment

type Template struct {
	SegmentSlug      string   `json:"segment_slug,omitempty"`
	Segments         []string `json:"segments,omitempty"`
	UserID           int      `json:"user_id,omitempty"`
	AssignSegments   []string `json:"assign_segments,omitempty"`
	UnassignSegments []string `json:"unassign_segments,omitempty"`
	Fraction         int      `json:"fraction,omitempty"`
	TTL              int      `json:"ttl"`
}

type RequestUserID struct {
	UserID int `json:"user_id"`
}

type RequestSegmentSlug struct {
	SegmentSlug string `json:"segment_slug"`
	Fraction    int    `json:"fraction"`
}

type RequestUpdateSegments struct {
	UserID           int      `json:"user_id"`
	AssignSegments   []string `json:"assign_segments"`
	UnassignSegments []string `json:"unassign_segments"`
	TTL              int      `json:"ttl"`
}

type UserSegments struct {
	UserID   int      `json:"user_id"`
	Segments []string `json:"segments"`
}
