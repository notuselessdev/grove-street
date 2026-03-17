#!/bin/bash

# Configuration
BG_COLOR="#0a0a0f"
BG_GRADIENT="#1a1b26"
GREEN_ACCENT="#00FF41"
ORANGE_BADGE="#e05010"
FONT_BOLD="/System/Library/Fonts/SFNS.ttf"
FONT_REGULAR="/System/Library/Fonts/SFNS.ttf"

# Card dimensions — wide enough for title + badge on same row
CARD_W=520
CARD_H=82
AVATAR=56
TEXT_X=84        # text left edge (after avatar + padding)
BADGE_W=104      # badge width
BADGE_H=20
BADGE_X=$((CARD_W - BADGE_W - 12))   # right-aligned: 404
TITLE_Y=29       # title baseline
BADGE_TOP=$(( TITLE_Y - BADGE_H + 4 ))  # align badge vertically with title: 13
BADGE_BOT=$(( BADGE_TOP + BADGE_H ))     # 33
BADGE_TEXT_Y=$(( BADGE_TOP + 14 ))       # badge text baseline: 27
SUBTITLE_Y=58

# 1. Create Base Background with Gradient
magick -size 1200x630 radial-gradient:"$BG_GRADIENT-$BG_COLOR" background.png

# 2. Add Green Glow at the bottom
magick background.png \
  \( -size 1200x300 radial-gradient:"rgba(0,255,65,0.15)-transparent" -gravity south \) \
  -composite bg_glow.png

# 3. Create Notification Card with rounded corners
magick -size ${CARD_W}x${CARD_H} canvas:none \
  -fill "rgba(30,30,35,0.95)" -draw "roundrectangle 0,0 ${CARD_W},${CARD_H} 16,16" \
  -stroke "rgba(255,255,255,0.1)" -strokewidth 1 -fill none \
  -draw "roundrectangle 0,0 $((CARD_W-1)),$((CARD_H-1)) 16,16" \
  card_bg.png

# 4. Prepare CJ Avatar (56x56 with rounded corners)
magick -size ${AVATAR}x${AVATAR} xc:none \
  -fill white -draw "roundrectangle 0,0 ${AVATAR},${AVATAR} 10,10" \
  /tmp/avatar_mask.png

magick assets/icon.png -resize ${AVATAR}x${AVATAR}^ -gravity Center -extent ${AVATAR}x${AVATAR} \
  /tmp/avatar_mask.png -alpha off -compose CopyOpacity -composite avatar.png

# 5. Build card: avatar + title (left) + badge (top-right) + subtitle
AVATAR_PAD=$(( (CARD_H - AVATAR) / 2 ))  # vertical center: 13
magick card_bg.png \
  avatar.png -geometry +${AVATAR_PAD}+${AVATAR_PAD} -composite \
  -font "$FONT_BOLD" -pointsize 15 -fill white \
  -annotate +${TEXT_X}+${TITLE_Y} "Carl Johnson in grove-street" \
  -fill "$ORANGE_BADGE" \
  -draw "roundrectangle ${BADGE_X},${BADGE_TOP} $((BADGE_X+BADGE_W)),${BADGE_BOT} 10,10" \
  -font "$FONT_BOLD" -pointsize 11 -fill white \
  -annotate +$((BADGE_X+10))+${BADGE_TEXT_Y} "Input Required" \
  -font "$FONT_REGULAR" -pointsize 13 -fill "#AAAAAA" \
  -annotate +${TEXT_X}+${SUBTITLE_Y} "So you think im a punk do you" \
  notification.png

# 6. Center notification horizontally on background
NOTIF_X=$(( (1200 - CARD_W) / 2 ))
magick bg_glow.png notification.png -geometry +${NOTIF_X}+110 -composite banner_step1.png

# 7. Add Headlines
magick banner_step1.png \
  -gravity North -font "$FONT_BOLD" -pointsize 46 -fill white \
  -annotate +0+285 "GTA San Andreas voice notifications" \
  -gravity North -font "$FONT_BOLD" -pointsize 46 -fill white \
  -annotate +0+345 "for AI coding agents." \
  -gravity North -font "$FONT_REGULAR" -pointsize 24 -fill "$GREEN_ACCENT" \
  -annotate +0+435 "Stop babysitting your terminal — let CJ watch it for you." \
  -fill "$GREEN_ACCENT" -draw "roundrectangle 475,510 725,514 3,3" \
  banner.png

# Cleanup
rm background.png bg_glow.png card_bg.png avatar.png notification.png banner_step1.png /tmp/avatar_mask.png
