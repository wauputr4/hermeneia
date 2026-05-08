export type VideoScene = {
  duration_seconds: number;
  text: string;
  visual: string;
};

export type VideoContent = {
  template: string;
  aspect_ratio: "9:16";
  fps: number;
  scenes: VideoScene[];
  caption: string;
};
