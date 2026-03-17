import { loadFont } from "@remotion/google-fonts/Inter";
import {
  AbsoluteFill,
  Audio,
  Img,
  Sequence,
  interpolate,
  spring,
  staticFile,
  useCurrentFrame,
  useVideoConfig,
} from "remotion";

const { fontFamily } = loadFont("normal", {
  weights: ["400", "600", "700"],
  subsets: ["latin"],
});

// ─── Timing (seconds) ────────────────────────────────────────────────────────
const T_PROMPT_IN   = 0.3;   // prompt line fades in
const T_TYPE_START  = 0.6;   // start typing "claude"
const T_CHAR_DELAY  = 0.09;  // seconds per character
const CMD           = "claude";
const T_TYPE_END    = T_TYPE_START + CMD.length * T_CHAR_DELAY; // ~1.14s
const T_ENTER       = T_TYPE_END + 0.3;   // 1.44s — enter key
const T_INIT        = T_ENTER + 0.25;     // 1.69s — initializing line
const T_BOX         = T_ENTER + 0.55;     // 1.99s — welcome box
const T_GREETING    = T_BOX + 0.4;        // 2.39s — "Hi! How can I help..."
const T_CURSOR      = T_GREETING + 0.35;  // 2.74s — input cursor appears
const T_NOTIF       = T_CURSOR + 0.2;     // 2.94s — notification + audio
const T_NOTIF_EXIT  = T_NOTIF + 3.8;     // 6.74s — notification fades out
const TOTAL_S       = T_NOTIF_EXIT + 0.5; // 7.24s

// ─── Notification card (exact grove-notify.swift values × 1.5) ───────────────
const S  = 1.5;
const NW = Math.round(360 * S); // 540
const NH = Math.round(68  * S); // 102
const CR = Math.round(20  * S); // 30
const IS = Math.round(40  * S); // 60  icon size
const IX = Math.round(14  * S); // 21  icon x
const IY = Math.round(14  * S); // 21  icon y (=(68-40)/2)
const IR = Math.round(10  * S); // 15  icon corner radius
const TX = Math.round(64  * S); // 96  text x (14+40+10)
const TW = Math.round(282 * S); // 423 text width (360-64-14)
// CSS y coords (macOS y=0 is bottom → flip: cssTop = winH - macY - lineH)
const TITLE_TOP = Math.round(16 * S); // 24  (macOS baseY=16)
const SUB_TOP   = Math.round(36 * S); // 54  (macOS baseY+lineH1+gap=36)
const FT = Math.round(13.5 * S);      // 20  font: title
const FC = Math.round(10.5 * S);      // 16  font: category
const FS = Math.round(12   * S);      // 18  font: subtitle
const SHX = Math.round(30  * S);
const SHW = Math.round(300 * S);
const MARGIN = 18;

// ─── Terminal colours (Tokyo Night palette) ───────────────────────────────────
const C = {
  bg:      "#1a1b26",
  titleBg: "#16161e",
  dim:     "#565f89",
  white:   "#c0caf5",
  green:   "#9ece6a",
  cyan:    "#7dcfff",
  yellow:  "#e0af68",
  prompt:  "#9ece6a",
  cmd:     "#c0caf5",
};

// ─── Helpers ─────────────────────────────────────────────────────────────────
const reveal = (
  frame: number,
  fps: number,
  atSecond: number,
  durationS = 0.25
) =>
  interpolate(frame, [atSecond * fps, (atSecond + durationS) * fps], [0, 1], {
    extrapolateLeft: "clamp",
    extrapolateRight: "clamp",
  });

const typewriter = (frame: number, fps: number, text: string, startS: number) => {
  const chars = Math.floor((frame / fps - startS) / T_CHAR_DELAY);
  return text.slice(0, Math.max(0, Math.min(chars, text.length)));
};

// ─── Blinking cursor ─────────────────────────────────────────────────────────
const Cursor: React.FC<{ visible: boolean; color?: string }> = ({
  visible,
  color = C.white,
}) => {
  const frame = useCurrentFrame();
  const blink = Math.floor(frame / 30) % 2 === 0;
  return (
    <span
      style={{
        display: "inline-block",
        width: 9,
        height: "1.1em",
        background: visible && blink ? color : "transparent",
        verticalAlign: "text-bottom",
        marginLeft: 1,
      }}
    />
  );
};

// ─── macOS terminal window ────────────────────────────────────────────────────
const MacTerminal: React.FC<{ children: React.ReactNode; width: number; height: number }> = ({
  children,
  width,
  height,
}) => (
  <div
    style={{
      width,
      height,
      borderRadius: 12,
      overflow: "hidden",
      boxShadow: "0 32px 80px rgba(0,0,0,0.75)",
      display: "flex",
      flexDirection: "column",
      fontFamily: "'Courier New', 'Menlo', monospace",
    }}
  >
    {/* Title bar */}
    <div
      style={{
        background: C.titleBg,
        height: 38,
        display: "flex",
        alignItems: "center",
        paddingLeft: 14,
        flexShrink: 0,
        gap: 8,
        borderBottom: `1px solid rgba(255,255,255,0.06)`,
      }}
    >
      <div style={{ width: 13, height: 13, borderRadius: "50%", background: "#ff5f57" }} />
      <div style={{ width: 13, height: 13, borderRadius: "50%", background: "#ffbd2e" }} />
      <div style={{ width: 13, height: 13, borderRadius: "50%", background: "#28c840" }} />
      <span
        style={{
          flex: 1,
          textAlign: "center",
          color: C.dim,
          fontSize: 13,
          fontFamily,
          marginRight: 52, // offset for dots
        }}
      >
        Terminal — -zsh
      </span>
    </div>

    {/* Content */}
    <div
      style={{
        flex: 1,
        background: C.bg,
        padding: "18px 22px",
        overflow: "hidden",
      }}
    >
      {children}
    </div>
  </div>
);

// ─── Terminal content ─────────────────────────────────────────────────────────
const TerminalContent: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const typed      = typewriter(frame, fps, CMD, T_TYPE_START);
  const isTyping   = frame / fps >= T_TYPE_START && frame / fps < T_TYPE_END + 0.1;
  const afterEnter = frame / fps >= T_ENTER + 0.1;
  const showInit   = frame / fps >= T_INIT;
  const showBox    = frame / fps >= T_BOX;
  const showGreet  = frame / fps >= T_GREETING;
  const showCursor = frame / fps >= T_CURSOR;

  const promptOpacity  = reveal(frame, fps, T_PROMPT_IN);
  const initOpacity    = reveal(frame, fps, T_INIT);
  const boxOpacity     = reveal(frame, fps, T_BOX, 0.2);
  const greetOpacity   = reveal(frame, fps, T_GREETING);
  const cursorOpacity  = reveal(frame, fps, T_CURSOR);

  const lineH = 24;
  const fs    = 15;

  return (
    <div style={{ fontSize: fs, lineHeight: `${lineH}px`, color: C.white }}>

      {/* Prompt line */}
      <div style={{ opacity: promptOpacity, display: "flex", alignItems: "center" }}>
        <span style={{ color: C.prompt }}>~/projects/my-app</span>
        <span style={{ color: C.dim }}> % </span>
        <span style={{ color: C.cmd }}>{typed}</span>
        {isTyping && <Cursor visible color={C.white} />}
      </div>

      {/* After enter: blank + output */}
      {afterEnter && (
        <>
          <div style={{ height: lineH }} />

          {showInit && (
            <div style={{ opacity: initOpacity, color: C.dim, marginBottom: 4 }}>
              Initializing Claude Code…
            </div>
          )}

          {showBox && (
            <div style={{ opacity: boxOpacity }}>
              <div style={{ color: C.dim }}>╭──────────────────────────────────────────────────────╮</div>
              <div>
                <span style={{ color: C.dim }}>│  </span>
                <span style={{ color: C.yellow }}>✻</span>
                <span style={{ color: C.white }}>  Claude Code</span>
                <span style={{ color: C.dim }}>                          v1.2.29  │</span>
              </div>
              <div>
                <span style={{ color: C.dim }}>│                                                      │</span>
              </div>
              <div>
                <span style={{ color: C.dim }}>│  </span>
                <span style={{ color: C.cyan }}>/help</span>
                <span style={{ color: C.dim }}> for help, </span>
                <span style={{ color: C.cyan }}>/status</span>
                <span style={{ color: C.dim }}> for your current setup     │</span>
              </div>
              <div style={{ color: C.dim }}>╰──────────────────────────────────────────────────────╯</div>
            </div>
          )}

          {showGreet && (
            <div style={{ opacity: greetOpacity, marginTop: lineH }}>
              <span style={{ color: C.yellow }}>✻</span>
              <span style={{ color: C.white }}> Hi! How can I help you today?</span>
            </div>
          )}

          {showCursor && (
            <div style={{ opacity: cursorOpacity, marginTop: 8, display: "flex", alignItems: "center" }}>
              <span style={{ color: C.green }}>❯ </span>
              <Cursor visible color={C.green} />
            </div>
          )}
        </>
      )}
    </div>
  );
};

// ─── Notification card ────────────────────────────────────────────────────────
const NotificationCard: React.FC<{
  opacity: number;
  translateY: number;
}> = ({ opacity, translateY }) => (
  <div
    style={{
      position: "absolute",
      top: MARGIN,
      right: MARGIN,
      width: NW,
      height: NH,
      borderRadius: CR,
      overflow: "hidden",
      opacity,
      transform: `translateY(${translateY}px)`,
    }}
  >
    {/* Liquid glass */}
    <div
      style={{
        position: "absolute",
        inset: 0,
        background: "linear-gradient(135deg, rgba(90,90,100,0.50) 0%, rgba(35,35,45,0.72) 100%)",
        backdropFilter: "blur(28px)",
        WebkitBackdropFilter: "blur(28px)",
        borderRadius: CR,
      }}
    />
    {/* Border */}
    <div style={{ position: "absolute", inset: 0, borderRadius: CR, border: "0.5px solid rgba(255,255,255,0.25)" }} />
    {/* Top shine */}
    <div style={{ position: "absolute", top: 1, left: SHX, width: SHW, height: 1, background: "rgba(255,255,255,0.18)", borderRadius: 1 }} />

    {/* Icon */}
    <div style={{ position: "absolute", left: IX, top: IY, width: IS, height: IS, borderRadius: IR, overflow: "hidden" }}>
      <Img src={staticFile("icon.png")} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
    </div>

    {/* Line 1: title + category */}
    <div style={{ position: "absolute", left: TX, top: TITLE_TOP, width: TW, height: 27, display: "flex", alignItems: "center", justifyContent: "space-between" }}>
      <span style={{ color: "rgba(255,255,255,0.95)", fontFamily, fontWeight: 700, fontSize: FT, lineHeight: 1, whiteSpace: "nowrap" }}>
        Carl Johnson in grove-street
      </span>
      <span style={{ color: "rgba(255,255,255,0.6)", fontFamily, fontWeight: 400, fontSize: FC, lineHeight: 1, whiteSpace: "nowrap", flexShrink: 0, marginLeft: 8 }}>
        Session Start
      </span>
    </div>

    {/* Line 2: subtitle */}
    <div style={{ position: "absolute", left: TX, top: SUB_TOP, width: TW, height: 24, display: "flex", alignItems: "center" }}>
      <span style={{ color: "rgba(255,255,255,0.55)", fontFamily, fontWeight: 400, fontSize: FS, lineHeight: 1, whiteSpace: "nowrap" }}>
        It&apos;s Carl, Carl Johnson!
      </span>
    </div>
  </div>
);

// ─── Main composition ─────────────────────────────────────────────────────────
export const NotificationVideo: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, width, height } = useVideoConfig();

  // Notification enter: spring slide-down (0.2s, matching Swift)
  const enterProgress = spring({ frame: frame - Math.round(T_NOTIF * fps), fps, config: { damping: 200 }, durationInFrames: Math.round(0.2 * fps) });
  const translateY    = interpolate(enterProgress, [0, 1], [-(NH + MARGIN), 0]);
  const enterOpacity  = interpolate(frame, [T_NOTIF * fps, (T_NOTIF + 0.2) * fps], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });

  // Notification exit: fade (0.3s, matching Swift)
  const exitOpacity = interpolate(frame, [T_NOTIF_EXIT * fps, (T_NOTIF_EXIT + 0.3) * fps], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
  const notifOpacity = Math.min(enterOpacity, exitOpacity);

  // Terminal window fade-in
  const terminalOpacity = reveal(frame, fps, 0.1, 0.3);

  // Terminal window: centred, fills most of the canvas
  const termW = Math.min(width - 60, 900);
  const termH = Math.min(height - 60, 400);
  const termX = (width - termW) / 2;
  const termY = (height - termH) / 2;

  return (
    <AbsoluteFill>
      {/* Blurred Grove Street background */}
      <Img src={staticFile("grove-bg.png")} style={{ width: "100%", height: "100%", objectFit: "cover" }} />
      <div style={{ position: "absolute", inset: 0, background: "rgba(0,0,0,0.72)" }} />

      {/* Terminal window */}
      <div style={{ position: "absolute", left: termX, top: termY, opacity: terminalOpacity }}>
        <MacTerminal width={termW} height={termH}>
          <TerminalContent />
        </MacTerminal>
      </div>

      {/* Notification */}
      <NotificationCard opacity={notifOpacity} translateY={translateY} />

      {/* Audio: plays exactly when notification slides in */}
      <Sequence from={Math.round(T_NOTIF * fps)}>
        <Audio src={staticFile("its_carl_carl_johnson.mp3")} />
      </Sequence>
    </AbsoluteFill>
  );
};

export const NOTIFICATION_VIDEO_DURATION_S = TOTAL_S;
