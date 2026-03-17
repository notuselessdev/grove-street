#!/usr/bin/env osascript -l JavaScript
// mac-overlay.js — Native macOS-style notification overlay for Grove Street
// Usage: osascript -l JavaScript mac-overlay.js <title> <message> <icon_path> <dismiss_seconds> [bundle_id]
//
// Apple-style notification: top-right, rounded, translucent dark background.
// Click to focus the target app. Auto-dismisses.

ObjC.import('Cocoa');

function run(argv) {
  var title    = argv[0] || 'Grove Street';
  var message  = argv[1] || '';
  var iconPath = argv[2] || '';
  var dismiss  = argv[3] !== undefined ? parseFloat(argv[3]) : 4;
  if (isNaN(dismiss)) dismiss = 4;
  var bundleId = argv[4] || '';

  var winWidth = 360, winHeight = 64;

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

    // Dark translucent background — Apple style
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

    // Visual effect view for native blur
    var effectView = $.NSVisualEffectView.alloc.initWithFrame(
      $.NSMakeRect(0, 0, winWidth, winHeight)
    );
    effectView.setMaterial($.NSVisualEffectMaterialHUDWindow);
    effectView.setBlendingMode($.NSVisualEffectBlendingModeBehindWindow);
    effectView.setState($.NSVisualEffectStateActive);
    effectView.wantsLayer = true;
    effectView.layer.cornerRadius = 16;
    effectView.layer.masksToBounds = true;
    contentView.addSubview(effectView);

    var textX = 14, textWidth = winWidth - 28;

    // Icon — rounded
    if (iconPath !== '' && $.NSFileManager.defaultManager.fileExistsAtPath(iconPath)) {
      var iconImage = $.NSImage.alloc.initWithContentsOfFile(iconPath);
      if (iconImage && !iconImage.isNil()) {
        var iconSize = 40;
        var iconView = $.NSImageView.alloc.initWithFrame(
          $.NSMakeRect(14, (winHeight - iconSize) / 2, iconSize, iconSize)
        );
        iconView.setImage(iconImage);
        iconView.setImageScaling($.NSImageScaleProportionallyUpOrDown);
        iconView.wantsLayer = true;
        iconView.layer.cornerRadius = 8;
        iconView.layer.masksToBounds = true;
        effectView.addSubview(iconView);
        textX = 14 + iconSize + 10;
        textWidth = winWidth - textX - 14;
      }
    }

    // Title
    var titleFont = $.NSFont.boldSystemFontOfSize(13);
    var titleHeight = 17;
    var titleY = message ? winHeight - titleHeight - 14 : (winHeight - titleHeight) / 2;
    var titleLabel = $.NSTextField.alloc.initWithFrame(
      $.NSMakeRect(textX, titleY, textWidth, titleHeight)
    );
    titleLabel.setStringValue($(title));
    titleLabel.setBezeled(false);
    titleLabel.setDrawsBackground(false);
    titleLabel.setEditable(false);
    titleLabel.setSelectable(false);
    titleLabel.setTextColor($.NSColor.whiteColor);
    titleLabel.setFont(titleFont);
    titleLabel.setLineBreakMode($.NSLineBreakByTruncatingTail);
    titleLabel.cell.setWraps(false);
    effectView.addSubview(titleLabel);

    // Subtitle
    if (message) {
      var msgFont = $.NSFont.systemFontOfSize(12);
      var msgLabel = $.NSTextField.alloc.initWithFrame(
        $.NSMakeRect(textX, titleY - 18, textWidth, 16)
      );
      msgLabel.setStringValue($(message));
      msgLabel.setBezeled(false);
      msgLabel.setDrawsBackground(false);
      msgLabel.setEditable(false);
      msgLabel.setSelectable(false);
      msgLabel.setTextColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.6));
      msgLabel.setFont(msgFont);
      msgLabel.setLineBreakMode($.NSLineBreakByTruncatingTail);
      msgLabel.cell.setWraps(false);
      effectView.addSubview(msgLabel);
    }

    // App name label top-right (like native notifications show the app name)
    var appFont = $.NSFont.systemFontOfSize(11);
    var appLabel = $.NSTextField.alloc.initWithFrame(
      $.NSMakeRect(winWidth - 110, winHeight - 17, 96, 14)
    );
    appLabel.setStringValue($('GROVE STREET'));
    appLabel.setBezeled(false);
    appLabel.setDrawsBackground(false);
    appLabel.setEditable(false);
    appLabel.setSelectable(false);
    appLabel.setTextColor($.NSColor.colorWithSRGBRedGreenBlueAlpha(1, 1, 1, 0.35));
    appLabel.setAlignment($.NSTextAlignmentRight);
    appLabel.setFont(appFont);
    effectView.addSubview(appLabel);

    // Transparent click button
    var btn = $.NSButton.alloc.initWithFrame($.NSMakeRect(0, 0, winWidth, winHeight));
    btn.setTitle($(''));
    btn.setBordered(false);
    btn.setTransparent(true);
    btn.setTarget(clickHandler);
    btn.setAction('handleClick');
    effectView.addSubview(btn);

    win.orderFrontRegardless;

    // Fade in
    win.animator.setAlphaValue(1.0);
  }

  // Auto-dismiss with fade out
  if (dismiss > 0) {
    // Fade out slightly before dismiss
    var fadeTime = Math.min(0.5, dismiss * 0.2);
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
