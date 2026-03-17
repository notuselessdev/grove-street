#!/usr/bin/env swift
// grove-notify.swift — Native macOS notification overlay for Grove Street
// Usage: grove-notify <sender> <phrase> <icon_path> <dismiss_seconds> <bundle_id> <project_name> <position> <slot_index> <slot_dir> [category_label]
//
// Positions: top-left, top-center, top-right, bottom-left, bottom-center, bottom-right, center

import Cocoa

// MARK: - Args

let args = CommandLine.arguments.dropFirst().map { $0 }
let senderName  = args.count > 0 ? args[0] : "Carl Johnson"
let phrase       = args.count > 1 ? args[1] : ""
let iconPath     = args.count > 2 ? args[2] : ""
let dismiss      = args.count > 3 ? Double(args[3]) ?? 4 : 4
let bundleId     = args.count > 4 ? args[4] : ""
let projectName  = args.count > 5 ? args[5] : "grove-street"
let position     = args.count > 6 ? args[6] : "top-right"
var slotIndex       = args.count > 7 ? Int(args[7]) ?? 0 : 0
let slotDir         = args.count > 8 ? args[8] : ""
let categoryLabel   = args.count > 9 ? args[9] : ""

let winWidth: CGFloat = 360
let winHeight: CGFloat = 68
let cornerR: CGFloat = 20
let margin: CGFloat = 12
let myPid = ProcessInfo.processInfo.processIdentifier

// MARK: - Slot management

func isSlotOccupied(_ idx: Int) -> Bool {
    guard !slotDir.isEmpty else { return true }
    let path = "\(slotDir)/\(idx).lock"
    guard FileManager.default.fileExists(atPath: path),
          let contents = try? String(contentsOfFile: path, encoding: .utf8) else {
        return false
    }
    guard let pid = Int32(contents.trimmingCharacters(in: .whitespacesAndNewlines)) else {
        return false
    }
    if pid == myPid { return true }
    return kill(pid, 0) == 0
}

func moveToSlot(_ newIdx: Int) {
    guard !slotDir.isEmpty else { return }
    let oldPath = "\(slotDir)/\(slotIndex).lock"
    let newPath = "\(slotDir)/\(newIdx).lock"
    try? "\(myPid)".write(toFile: newPath, atomically: true, encoding: .utf8)
    try? FileManager.default.removeItem(atPath: oldPath)
    slotIndex = newIdx
}

func cleanupSlot() {
    guard !slotDir.isEmpty else { return }
    try? FileManager.default.removeItem(atPath: "\(slotDir)/\(slotIndex).lock")
}

// MARK: - Position calculation

func calcOrigin(visibleFrame vf: NSRect, slot: Int) -> NSPoint {
    let stackOffset = CGFloat(slot) * (winHeight + margin)
    var x: CGFloat, y: CGFloat

    switch position {
    case "top-left":
        x = vf.origin.x + margin
        y = vf.origin.y + vf.size.height - winHeight - margin - stackOffset
    case "top-center":
        x = vf.origin.x + (vf.size.width - winWidth) / 2
        y = vf.origin.y + vf.size.height - winHeight - margin - stackOffset
    case "top-right":
        x = vf.origin.x + vf.size.width - winWidth - margin
        y = vf.origin.y + vf.size.height - winHeight - margin - stackOffset
    case "bottom-left":
        x = vf.origin.x + margin
        y = vf.origin.y + margin + stackOffset
    case "bottom-center":
        x = vf.origin.x + (vf.size.width - winWidth) / 2
        y = vf.origin.y + margin + stackOffset
    case "bottom-right":
        x = vf.origin.x + vf.size.width - winWidth - margin
        y = vf.origin.y + margin + stackOffset
    case "center":
        x = vf.origin.x + (vf.size.width - winWidth) / 2
        let halfOffset = stackOffset / 2
        y = vf.origin.y + (vf.size.height - winHeight) / 2 + (slot % 2 == 0 ? -halfOffset : halfOffset)
    default:
        x = vf.origin.x + vf.size.width - winWidth - margin
        y = vf.origin.y + vf.size.height - winHeight - margin - stackOffset
    }
    return NSPoint(x: x, y: y)
}

// MARK: - App setup

let app = NSApplication.shared
app.setActivationPolicy(.accessory)

var windows: [(window: NSWindow, screen: NSScreen)] = []

for screen in NSScreen.screens {
    let vf = screen.visibleFrame
    let origin = calcOrigin(visibleFrame: vf, slot: slotIndex)
    let frame = NSRect(x: origin.x, y: origin.y, width: winWidth, height: winHeight)

    let win = NSWindow(contentRect: frame, styleMask: .borderless, backing: .buffered, defer: false)
    win.backgroundColor = .clear
    win.isOpaque = false
    win.alphaValue = 0
    win.level = .statusBar
    win.collectionBehavior = [.canJoinAllSpaces, .stationary]
    win.hasShadow = true

    let contentView = win.contentView!
    contentView.wantsLayer = true

    // Liquid glass blur
    let effectView = NSVisualEffectView(frame: NSRect(x: 0, y: 0, width: winWidth, height: winHeight))
    effectView.material = .hudWindow
    effectView.blendingMode = .behindWindow
    effectView.state = .active
    effectView.alphaValue = 0.82
    effectView.wantsLayer = true
    effectView.layer?.cornerRadius = cornerR
    effectView.layer?.masksToBounds = true
    contentView.addSubview(effectView)

    // Glass edge border
    let borderBox = NSBox(frame: NSRect(x: 0, y: 0, width: winWidth, height: winHeight))
    borderBox.boxType = .custom
    borderBox.cornerRadius = cornerR
    borderBox.borderWidth = 0.5
    borderBox.borderColor = NSColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.25)
    borderBox.fillColor = .clear
    borderBox.isTransparent = false
    effectView.addSubview(borderBox)

    // Top edge shine
    let edgeShine = NSBox(frame: NSRect(x: 30, y: winHeight - 1.5, width: winWidth - 60, height: 0.5))
    edgeShine.boxType = .custom
    edgeShine.cornerRadius = 0.25
    edgeShine.borderWidth = 0
    edgeShine.borderColor = .clear
    edgeShine.fillColor = NSColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.18)
    edgeShine.isTransparent = false
    effectView.addSubview(edgeShine)

    var textX: CGFloat = 14
    var textWidth = winWidth - 28

    // Icon
    if !iconPath.isEmpty, FileManager.default.fileExists(atPath: iconPath),
       let iconImage = NSImage(contentsOfFile: iconPath) {
        let iconSz: CGFloat = 40
        let iconView = NSImageView(frame: NSRect(x: 14, y: (winHeight - iconSz) / 2, width: iconSz, height: iconSz))
        iconView.image = iconImage
        iconView.imageScaling = .scaleProportionallyUpOrDown
        iconView.wantsLayer = true
        iconView.layer?.cornerRadius = 10
        iconView.layer?.masksToBounds = true
        effectView.addSubview(iconView)
        textX = 14 + iconSz + 10
        textWidth = winWidth - textX - 14
    }

    // Text layout
    let lineHeight1: CGFloat = 18
    let lineHeight2: CGFloat = 16
    let gap: CGFloat = 2
    let totalHeight = lineHeight1 + gap + lineHeight2
    let baseY = (winHeight - totalHeight) / 2

    // Line 1 left: sender in project
    let senderLabel = NSTextField(frame: NSRect(x: textX, y: baseY + lineHeight2 + gap, width: textWidth, height: lineHeight1))
    senderLabel.stringValue = "\(senderName) in \(projectName)"
    senderLabel.isBezeled = false
    senderLabel.drawsBackground = false
    senderLabel.isEditable = false
    senderLabel.isSelectable = false
    senderLabel.textColor = NSColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.95)
    senderLabel.font = .boldSystemFont(ofSize: 13.5)
    senderLabel.lineBreakMode = .byTruncatingTail
    effectView.addSubview(senderLabel)

    // Line 1 right: category label (vertically centered within the sender line)
    if !categoryLabel.isEmpty {
        let catFont = NSFont.systemFont(ofSize: 10.5)
        let catH: CGFloat = 14
        let catY = baseY + lineHeight2 + gap + (lineHeight1 - catH) / 2
        let catLabel = NSTextField(frame: NSRect(x: textX, y: catY, width: textWidth, height: catH))
        catLabel.stringValue = categoryLabel
        catLabel.isBezeled = false
        catLabel.drawsBackground = false
        catLabel.isEditable = false
        catLabel.isSelectable = false
        catLabel.textColor = NSColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.6)
        catLabel.font = catFont
        catLabel.alignment = .right
        catLabel.lineBreakMode = .byClipping
        effectView.addSubview(catLabel)
    }

    // Line 2: phrase
    if !phrase.isEmpty {
        let msgLabel = NSTextField(frame: NSRect(x: textX, y: baseY, width: textWidth, height: lineHeight2))
        msgLabel.stringValue = phrase
        msgLabel.isBezeled = false
        msgLabel.drawsBackground = false
        msgLabel.isEditable = false
        msgLabel.isSelectable = false
        msgLabel.textColor = NSColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.55)
        msgLabel.font = .systemFont(ofSize: 12)
        msgLabel.lineBreakMode = .byTruncatingTail
        msgLabel.cell?.wraps = false
        effectView.addSubview(msgLabel)
    }

    // Click button (transparent look, but NOT .isTransparent which disables hit-testing)
    let btn = NSButton(frame: NSRect(x: 0, y: 0, width: winWidth, height: winHeight))
    btn.title = ""
    btn.isBordered = false
    btn.isTransparent = false
    btn.alphaValue = 0.001
    btn.target = nil
    btn.action = #selector(NSApplication.terminate(_:))
    effectView.addSubview(btn)

    win.orderFrontRegardless()
    NSAnimationContext.runAnimationGroup { ctx in
        ctx.duration = 0.2
        win.animator().alphaValue = 1.0
    }

    windows.append((window: win, screen: screen))
}

// MARK: - Click handler to focus app

class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationWillTerminate(_ notification: Notification) {
        if !bundleId.isEmpty {
            if let targetApp = NSRunningApplication.runningApplications(withBundleIdentifier: bundleId).first {
                targetApp.activate()
            }
        }
        cleanupSlot()
    }
}

let delegate = AppDelegate()
app.delegate = delegate

// MARK: - Reflow timer

if !slotDir.isEmpty {
    Timer.scheduledTimer(withTimeInterval: 0.5, repeats: true) { _ in
        for s in 0..<slotIndex {
            if !isSlotOccupied(s) {
                moveToSlot(s)
                NSAnimationContext.runAnimationGroup { ctx in
                    ctx.duration = 0.25
                    ctx.allowsImplicitAnimation = true
                    for entry in windows {
                        let newOrigin = calcOrigin(visibleFrame: entry.screen.visibleFrame, slot: slotIndex)
                        var newFrame = entry.window.frame
                        newFrame.origin = newOrigin
                        entry.window.animator().setFrame(newFrame, display: true)
                    }
                }
                break
            }
        }
    }
}

// MARK: - Auto-dismiss

if dismiss > 0 {
    Timer.scheduledTimer(withTimeInterval: dismiss, repeats: false) { _ in
        NSAnimationContext.runAnimationGroup({ ctx in
            ctx.duration = 0.3
            for entry in windows {
                entry.window.animator().alphaValue = 0
            }
        }, completionHandler: {
            app.terminate(nil)
        })
    }
}

app.run()
