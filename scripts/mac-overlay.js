#!/usr/bin/env osascript -l JavaScript
// mac-overlay.js — Liquid glass notification overlay for Grove Street
// Usage: osascript -l JavaScript mac-overlay.js <sender> <phrase> <icon_path> <dismiss_seconds> [bundle_id] [project_name]
//
// Chat notification style: "Carl Johnson in project-name" + voice line phrase.
// Liquid glass aesthetic: translucent blur, glass edge border, top edge highlight.
// Click to focus the target app. Auto-dismisses.

ObjC.import('Cocoa');

function run(argv) {
  var senderName  = argv[0] || 'Carl Johnson';
  var phrase      = argv[1] || '';
  var iconPath    = argv[2] || '';
  var dismiss     = argv[3] !== undefined ? parseFloat(argv[3]) : 4;
  if (isNaN(dismiss)) dismiss = 4;
  var bundleId    = argv[4] || '';
  var projectName = argv[5] || 'grove-street';

  var winWidth = 360, winHeight = 68;
  var cornerR = 20;

  $.NSApplication.sharedApplication;
  $.NSApp.setActivationPolicy($.NSApplicationActivationPolicyAccessory);

  // Click handler
  ObjC.registerSubclass({
    name: 'GroveClickHandler',
    superclass: 'NSObject',
    methods: {
      'handleClick': {
        types: ['void', []],
        implementation: function() {
          if (bundleId) {
            var ws = $.NSWorkspace.sharedWorkspace;
            var apps = ws.runningApplications;
            var count = apps.count;
            for (var i = 0; i < count; i++) {
              var app = apps.objectAtIndex(i);
              var bid = app.bundleIdentifier;
              if (!bid.isNil() && bid.js === bundleId) {
                app.activateWithOptions($.NSApplicationActivateIgnoringOtherApps);
                break;
              }
            }
          }
          $.NSApp.terminate(null);
        }
      }
    }
  });
  var clickHandler = $.GroveClickHandler.alloc.init;


  var screens = $.NSScreen.screens;
  var screenCount = screens.count;

  for (var i = 0; i < screenCount; i++) {
    var screen = screens.objectAtIndex(i);
    var visibleFrame = screen.visibleFrame;

    // Top-right, like native macOS notifications
    var margin = 12;
    var x = visibleFrame.origin.x + visibleFrame.size.width - winWidth - margin;
    var y = visibleFrame.origin.y + visibleFrame.size.height - winHeight - margin;
    var frame = $.NSMakeRect(x, y, winWidth, winHeight);

    var win = $.NSWindow.alloc.initWithContentRectStyleMaskBackingDefer(
      frame,
      $.NSWindowStyleMaskBorderless,
      $.NSBackingStoreBuffered,
      false
    );

    win.setBackgroundColor($.NSColor.clearColor);
    win.setOpaque(false);
    win.setAlphaValue(0.0);
    win.setLevel($.NSStatusWindowLevel);
    win.setCollectionBehavior(
      $.NSWindowCollectionBehaviorCanJoinAllSpaces |
      $.NSWindowCollectionBehaviorStationary
    );
    win.setHasShadow(true);


    var contentView = win.contentView;
    contentView.wantsLayer = true;

    // Liquid glass — translucent blur, slightly see-through
    var effectView = $.NSVisualEffectView.alloc.initWithFrame(
      $.NSMakeRect(0, 0, winWidth, winHeight)
    );
    effectView.setMaterial($.NSVisualEffectMaterialHUDWindow);
    effectView.setBlendingMode($.NSVisualEffectBlendingModeBehindWindow);
    effectView.setState($.NSVisualEffectStateActive);
    effectView.setAlphaValue(0.82);
    effectView.wantsLayer = true;
    effectView.layer.cornerRadius = cornerR;
    effectView.layer.masksToBounds = true;
    contentView.addSubview(effectView);

    // Glass edge border
    var borderBox = $.NSBox.alloc.initWithFrame(
      $.NSMakeRect(0, 0, winWidth, winHeight)
    );
    borderBox.setBoxType($.NSBoxCustom);
    borderBox.setCornerRadius(cornerR);
    borderBox.setBorderWidth(0.5);
    borderBox.setBorderColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.25));
    borderBox.setFillColor($.NSColor.clearColor);
    borderBox.setTransparent(false);
    effectView.addSubview(borderBox);

    // Top edge static highlight
    var edgeShine = $.NSBox.alloc.initWithFrame(
      $.NSMakeRect(30, winHeight - 1.5, winWidth - 60, 0.5)
    );
    edgeShine.setBoxType($.NSBoxCustom);
    edgeShine.setCornerRadius(0.25);
    edgeShine.setBorderWidth(0);
    edgeShine.setBorderColor($.NSColor.clearColor);
    edgeShine.setFillColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.18));
    edgeShine.setTransparent(false);
    effectView.addSubview(edgeShine);

    var textX = 14, textWidth = winWidth - 28;

    // Icon — rounded
    if (iconPath !== '' && $.NSFileManager.defaultManager.fileExistsAtPath(iconPath)) {
      var iconImage = $.NSImage.alloc.initWithContentsOfFile(iconPath);
      if (iconImage && !iconImage.isNil()) {
        var iconSz = 40;
        var iconView = $.NSImageView.alloc.initWithFrame(
          $.NSMakeRect(14, (winHeight - iconSz) / 2, iconSz, iconSz)
        );
        iconView.setImage(iconImage);
        iconView.setImageScaling($.NSImageScaleProportionallyUpOrDown);
        iconView.wantsLayer = true;
        iconView.layer.cornerRadius = 10;
        iconView.layer.masksToBounds = true;
        effectView.addSubview(iconView);
        textX = 14 + iconSz + 10;
        textWidth = winWidth - textX - 14;
      }
    }

    // Two-line chat-style layout, vertically centered
    var lineHeight1 = 18;
    var lineHeight2 = 16;
    var gap = 2;
    var totalHeight = lineHeight1 + gap + lineHeight2;
    var baseY = (winHeight - totalHeight) / 2;

    // Line 1: "Carl Johnson in project-name"
    var senderLabel = $.NSTextField.alloc.initWithFrame(
      $.NSMakeRect(textX, baseY + lineHeight2 + gap, textWidth, lineHeight1)
    );
    senderLabel.setStringValue($(senderName + ' in ' + projectName));
    senderLabel.setBezeled(false);
    senderLabel.setDrawsBackground(false);
    senderLabel.setEditable(false);
    senderLabel.setSelectable(false);
    senderLabel.setTextColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.95));
    senderLabel.setFont($.NSFont.boldSystemFontOfSize(13.5));
    senderLabel.setLineBreakMode($.NSLineBreakByTruncatingTail);
    effectView.addSubview(senderLabel);

    // Line 2: voice line phrase
    if (phrase) {
      var msgLabel = $.NSTextField.alloc.initWithFrame(
        $.NSMakeRect(textX, baseY, textWidth, lineHeight2)
      );
      msgLabel.setStringValue($(phrase));
      msgLabel.setBezeled(false);
      msgLabel.setDrawsBackground(false);
      msgLabel.setEditable(false);
      msgLabel.setSelectable(false);
      msgLabel.setTextColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.55));
      msgLabel.setFont($.NSFont.systemFontOfSize(12));
      msgLabel.setLineBreakMode($.NSLineBreakByTruncatingTail);
      msgLabel.cell.setWraps(false);
      effectView.addSubview(msgLabel);
    }

    // Transparent click button
    var btn = $.NSButton.alloc.initWithFrame($.NSMakeRect(0, 0, winWidth, winHeight));
    btn.setTitle($(''));
    btn.setBordered(false);
    btn.setTransparent(true);
    btn.setTarget(clickHandler);
    btn.setAction('handleClick');
    effectView.addSubview(btn);

    win.orderFrontRegardless;
    win.animator.setAlphaValue(1.0);

  }

  // Auto-dismiss
  if (dismiss > 0) {
    $.NSTimer.scheduledTimerWithTimeIntervalTargetSelectorUserInfoRepeats(
      dismiss,
      $.NSApp,
      'terminate:',
      null,
      false
    );
  }

  $.NSApp.run;
}
