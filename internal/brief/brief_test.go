package brief

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBriefJSONRoundTrip(t *testing.T) {
	want := Brief{
		Topic:           "AI agents in marketing",
		Angle:           "Practical adoption without replacing the team",
		Hook:            "AI agents are becoming the new marketing interns.",
		TargetAudience:  "Small business owners and marketing leads",
		Platform:        "instagram",
		ContentType:     "carousel",
		Tone:            "clear, optimistic, pragmatic",
		KeyPoints:       []string{"Use agents for research", "Keep humans in approval loops", "Measure workflow time saved"},
		VisualDirection: "Clean editorial slides with product UI accents",
		CTA:             "Save this workflow before planning your next campaign.",
		CaptionDraft:    "AI agents can help marketing teams move faster when the workflow stays human-led.",
		Hashtags:        []string{"#AI", "#Marketing", "#Automation"},
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	var got Brief
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestExampleBriefMatchesSchema(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "brief.ai-agents-carousel.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got Brief
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	if got.Topic == "" || got.Hook == "" || len(got.KeyPoints) == 0 || len(got.Hashtags) == 0 {
		t.Fatalf("example brief is incomplete: %#v", got)
	}
}
