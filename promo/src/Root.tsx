import { Composition, Still } from "remotion";
import { NotificationVideo } from "./NotificationVideo";
import { PromoBanner } from "./PromoBanner";

import { NOTIFICATION_VIDEO_DURATION_S } from "./NotificationVideo";

const FPS = 60;
const FRAMES = Math.round(NOTIFICATION_VIDEO_DURATION_S * FPS);

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Still id="PromoBanner" component={PromoBanner} width={1200} height={630} />

      {/* 16:9 — README embed / Twitter / YouTube */}
      <Composition
        id="NotificationDemo"
        component={NotificationVideo}
        durationInFrames={FRAMES}
        fps={FPS}
        width={1280}
        height={720}
      />

      {/* 1:1 — Instagram / LinkedIn */}
      <Composition
        id="NotificationDemoSquare"
        component={NotificationVideo}
        durationInFrames={FRAMES}
        fps={FPS}
        width={1080}
        height={1080}
      />

      {/* 9:16 — Reels / TikTok / Stories */}
      <Composition
        id="NotificationDemoVertical"
        component={NotificationVideo}
        durationInFrames={FRAMES}
        fps={FPS}
        width={1080}
        height={1920}
      />
    </>
  );
};
