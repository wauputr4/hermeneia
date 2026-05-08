package brief

// Brief is the structured content planning document used by the MVP workflow.
// It is stored as JSON in brief_versions.body_json and mirrored in run folders.
type Brief struct {
	Topic           string   `json:"topic"`
	Angle           string   `json:"angle"`
	Hook            string   `json:"hook"`
	TargetAudience  string   `json:"target_audience"`
	Platform        string   `json:"platform"`
	ContentType     string   `json:"content_type"`
	Tone            string   `json:"tone"`
	KeyPoints       []string `json:"key_points"`
	VisualDirection string   `json:"visual_direction"`
	CTA             string   `json:"cta"`
	CaptionDraft    string   `json:"caption_draft"`
	Hashtags        []string `json:"hashtags"`
}
