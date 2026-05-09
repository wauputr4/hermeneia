import React from "react";
import { Composition, registerRoot } from "remotion";
import { AINewsShort } from "./AINewsShort";
import type { VideoContent } from "./schema";

const defaultContent: VideoContent = {
  template: "video/ai-news-short",
  aspect_ratio: "9:16",
  fps: 30,
  scenes: [
    {
      duration_seconds: 3,
      text: "Hermeneia turns structured briefs into short video scenes.",
      visual: "Clean editorial motion layout",
    },
  ],
  caption: "Hermeneia video renderer contract preview.",
};

export const RemotionRoot = () => (
  <Composition
    id="AINewsShort"
    component={AINewsShort}
    durationInFrames={defaultContent.scenes.reduce(
      (frames, scene) => frames + scene.duration_seconds * defaultContent.fps,
      0,
    )}
    fps={defaultContent.fps}
    width={1080}
    height={1920}
    defaultProps={{ content: defaultContent }}
  />
);

registerRoot(RemotionRoot);
