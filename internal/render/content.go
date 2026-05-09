package render

import "github.com/wauputr4/hermeneia/internal/brief"

const (
	TemplateCarouselAINewsClean = "carousel/ai-news-clean"
	TemplateVideoAINewsShort    = "video/ai-news-short"
)

type OutputFile struct {
	Kind string
	Path string
}

type CarouselContent struct {
	Template string          `json:"template"`
	Slides   []CarouselSlide `json:"slides"`
	Caption  string          `json:"caption"`
	Hashtags []string        `json:"hashtags"`
}

type CarouselSlide struct {
	Type     string `json:"type"`
	Headline string `json:"headline"`
	Body     string `json:"body"`
}

type VideoContent struct {
	Template    string       `json:"template"`
	AspectRatio string       `json:"aspect_ratio"`
	FPS         int          `json:"fps"`
	Scenes      []VideoScene `json:"scenes"`
	Caption     string       `json:"caption"`
}

type VideoScene struct {
	DurationSeconds int    `json:"duration_seconds"`
	Text            string `json:"text"`
	Visual          string `json:"visual"`
}

func BuildCarouselContent(b brief.Brief, template string) CarouselContent {
	if template == "" {
		template = TemplateCarouselAINewsClean
	}
	slides := []CarouselSlide{
		{
			Type:     "cover",
			Headline: b.Hook,
			Body:     b.Angle,
		},
	}
	for _, point := range b.KeyPoints {
		slides = append(slides, CarouselSlide{
			Type:     "point",
			Headline: b.Topic,
			Body:     point,
		})
	}
	slides = append(slides, CarouselSlide{
		Type:     "closing",
		Headline: "Next step",
		Body:     b.CTA,
	})
	return CarouselContent{
		Template: template,
		Slides:   slides,
		Caption:  b.CaptionDraft,
		Hashtags: b.Hashtags,
	}
}

func BuildVideoContent(b brief.Brief, template string) VideoContent {
	if template == "" {
		template = TemplateVideoAINewsShort
	}
	scenes := []VideoScene{
		{
			DurationSeconds: 3,
			Text:            b.Hook,
			Visual:          b.VisualDirection,
		},
	}
	for _, point := range b.KeyPoints {
		scenes = append(scenes, VideoScene{
			DurationSeconds: 3,
			Text:            point,
			Visual:          b.VisualDirection,
		})
	}
	scenes = append(scenes, VideoScene{
		DurationSeconds: 3,
		Text:            b.CTA,
		Visual:          "closing call to action",
	})
	return VideoContent{
		Template:    template,
		AspectRatio: "9:16",
		FPS:         30,
		Scenes:      scenes,
		Caption:     b.CaptionDraft,
	}
}
