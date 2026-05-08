import React from "react";
import { AbsoluteFill, interpolate, useCurrentFrame, useVideoConfig } from "remotion";
import type { VideoContent } from "./schema";

export function AINewsShort({ content }: { content: VideoContent }) {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const sceneStarts = content.scenes.reduce<number[]>((starts, scene, index) => {
    starts[index] = index === 0 ? 0 : starts[index - 1] + content.scenes[index - 1].duration_seconds * fps;
    return starts;
  }, []);
  let sceneIndex = 0;
  for (let index = 0; index < sceneStarts.length; index += 1) {
    if (frame >= sceneStarts[index]) {
      sceneIndex = index;
    }
  }
  const scene = content.scenes[sceneIndex] ?? content.scenes[0];
  const progress = interpolate(frame - (sceneStarts[sceneIndex] ?? 0), [0, fps], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

  return (
    <AbsoluteFill
      style={{
        background: "#172326",
        color: "#f6f4ed",
        fontFamily: "ui-sans-serif, system-ui, sans-serif",
        padding: 96,
        justifyContent: "center",
      }}
    >
      <div style={{ color: "#e8583d", fontSize: 42, fontWeight: 800, marginBottom: 48 }}>
        HERMENEIA
      </div>
      <div
        style={{
          fontSize: 76,
          fontWeight: 800,
          lineHeight: 1.04,
          opacity: progress,
        }}
      >
        {scene.text}
      </div>
      <div style={{ color: "#8fd3d8", fontSize: 34, marginTop: 52 }}>{scene.visual}</div>
    </AbsoluteFill>
  );
}
