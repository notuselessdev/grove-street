import { loadFont } from "@remotion/google-fonts/Inter";
import { AbsoluteFill, Img, staticFile } from "remotion";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700"],
  subsets: ["latin"],
});

// Exact values from grove-notify.swift, scaled ×1.6 for promo clarity
// NOTE: macOS NSView has y=0 at bottom; CSS has y=0 at top — values are flipped below.
const SCALE = 1.6;
const WIN_W = Math.round(360 * SCALE); // 576
const WIN_H = Math.round(68 * SCALE);  // 109
const CORNER_R = Math.round(20 * SCALE); // 32
const ICON_SZ = Math.round(40 * SCALE);  // 64
const ICON_X = Math.round(14 * SCALE);   // 22
const ICON_Y = Math.round(14 * SCALE);   // 22 (=(68-40)/2)
const ICON_R = Math.round(10 * SCALE);   // 16
const TEXT_X = Math.round(64 * SCALE);   // 102  (14+40+10)
const TEXT_W = Math.round(282 * SCALE);  // 451  (360-64-14)
const LINE1_H = Math.round(18 * SCALE);  // 29
const LINE2_H = Math.round(16 * SCALE);  // 26
// CSS tops (converted from macOS bottom-origin):
// macOS baseY=16 → sender CSS top = (68-36)/2 = 16 → *SCALE = 26
// subtitle CSS top = 16+18+2 = 36 → *SCALE = 58
const TITLE_TOP = Math.round(16 * SCALE); // 26
const SUB_TOP   = Math.round(36 * SCALE); // 58
const FONT_TITLE = Math.round(13.5 * SCALE); // 22
const FONT_CAT   = Math.round(10.5 * SCALE); // 17
const FONT_SUB   = Math.round(12 * SCALE);   // 19
const SHINE_X = Math.round(30 * SCALE);          // 48
const SHINE_W = Math.round((360 - 60) * SCALE);  // 480

// Category label: right-aligned plain text (white@0.6), per Swift code
const CATEGORY = "Input Required";

const NotificationCard: React.FC = () => {
  return (
    <div
      style={{
        position: "relative",
        width: WIN_W,
        height: WIN_H,
        borderRadius: CORNER_R,
        overflow: "hidden",
      }}
    >
      {/* Liquid glass background — simulates NSVisualEffectView .hudWindow */}
      <div
        style={{
          position: "absolute",
          inset: 0,
          background:
            "linear-gradient(135deg, rgba(80,80,90,0.55) 0%, rgba(40,40,50,0.72) 100%)",
          backdropFilter: "blur(24px)",
          WebkitBackdropFilter: "blur(24px)",
          borderRadius: CORNER_R,
        }}
      />

      {/* Glass edge border — white@0.25, 0.5px */}
      <div
        style={{
          position: "absolute",
          inset: 0,
          borderRadius: CORNER_R,
          border: "0.5px solid rgba(255,255,255,0.25)",
          pointerEvents: "none",
        }}
      />

      {/* Top edge shine */}
      <div
        style={{
          position: "absolute",
          top: 1,
          left: SHINE_X,
          width: SHINE_W,
          height: 1,
          background: "rgba(255,255,255,0.18)",
          borderRadius: 1,
        }}
      />

      {/* Icon */}
      <div
        style={{
          position: "absolute",
          left: ICON_X,
          top: ICON_Y,
          width: ICON_SZ,
          height: ICON_SZ,
          borderRadius: ICON_R,
          overflow: "hidden",
        }}
      >
        <Img
          src={staticFile("icon.png")}
          style={{ width: "100%", height: "100%", objectFit: "cover" }}
        />
      </div>

      {/* Line 1: title (left) + category (right) */}
      <div
        style={{
          position: "absolute",
          left: TEXT_X,
          top: TITLE_TOP,
          width: TEXT_W,
          height: LINE1_H,
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
        }}
      >
        <span
          style={{
            color: "rgba(255,255,255,0.95)",
            fontFamily,
            fontWeight: 700,
            fontSize: FONT_TITLE,
            lineHeight: 1,
            whiteSpace: "nowrap",
          }}
        >
          Carl Johnson in grove-street
        </span>
        <span
          style={{
            color: "rgba(255,255,255,0.6)",
            fontFamily,
            fontWeight: 400,
            fontSize: FONT_CAT,
            lineHeight: 1,
            whiteSpace: "nowrap",
            flexShrink: 0,
            marginLeft: 8,
          }}
        >
          {CATEGORY}
        </span>
      </div>

      {/* Line 2: subtitle */}
      <div
        style={{
          position: "absolute",
          left: TEXT_X,
          top: SUB_TOP,
          width: TEXT_W,
          height: LINE2_H,
          display: "flex",
          alignItems: "center",
        }}
      >
        <span
          style={{
            color: "rgba(255,255,255,0.55)",
            fontFamily,
            fontWeight: 400,
            fontSize: FONT_SUB,
            lineHeight: 1,
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          So you think im a punk do you
        </span>
      </div>
    </div>
  );
};

export const PromoBanner: React.FC = () => {
  return (
    <AbsoluteFill>
      {/* Background: Grove Street at night, fills frame */}
      <Img
        src={staticFile("grove-bg.png")}
        style={{ width: "100%", height: "100%", objectFit: "cover" }}
      />

      {/* Bottom gradient for text legibility */}
      <div
        style={{
          position: "absolute",
          inset: 0,
          background:
            "linear-gradient(to bottom, rgba(0,0,0,0.08) 0%, rgba(0,0,0,0.0) 25%, rgba(0,0,0,0.55) 55%, rgba(0,0,0,0.80) 100%)",
        }}
      />

      {/* Centered layout */}
      <AbsoluteFill
        style={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: 40,
          padding: "0 80px",
        }}
      >
        <NotificationCard />

        <div
          style={{
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
            gap: 16,
            textAlign: "center",
          }}
        >
          <h1
            style={{
              color: "white",
              fontFamily,
              fontSize: 40,
              fontWeight: 700,
              margin: 0,
              lineHeight: 1.2,
              maxWidth: 800,
              textShadow: "0 2px 12px rgba(0,0,0,0.6)",
            }}
          >
            GTA San Andreas voice notifications for AI coding agents.
          </h1>
          <p
            style={{
              color: "rgba(210, 210, 210, 0.92)",
              fontFamily,
              fontSize: 20,
              fontWeight: 400,
              margin: 0,
              textShadow: "0 1px 8px rgba(0,0,0,0.5)",
            }}
          >
            Stop babysitting your terminal — let CJ watch it for you.
          </p>
        </div>
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
