#!/usr/bin/env python3
"""grove-notify.py — Native Linux notification overlay for Grove Street.

Usage: grove-notify.py <sender> <phrase> <icon_path> <dismiss_seconds> <bundle_id> <project_name> <position> <slot_index> <slot_dir>

Positions: top-left, top-center, top-right, bottom-left, bottom-center, bottom-right, center
"""

import os
import sys
import signal

import gi
gi.require_version("Gtk", "3.0")
gi.require_version("Gdk", "3.0")
from gi.repository import Gtk, Gdk, GLib, GdkPixbuf, Pango

# --- Args ---

args = sys.argv[1:]
sender_name = args[0] if len(args) > 0 else "Carl Johnson"
phrase = args[1] if len(args) > 1 else ""
icon_path = args[2] if len(args) > 2 else ""
dismiss_secs = float(args[3]) if len(args) > 3 else 4.0
bundle_id = args[4] if len(args) > 4 else ""  # unused on Linux
project_name = args[5] if len(args) > 5 else "grove-street"
position = args[6] if len(args) > 6 else "top-right"
slot_index = int(args[7]) if len(args) > 7 else 0
slot_dir = args[8] if len(args) > 8 else ""

WIN_WIDTH = 360
WIN_HEIGHT = 68
CORNER_R = 16
MARGIN = 12
MY_PID = os.getpid()

# --- Slot management ---


def is_slot_occupied(idx):
    if not slot_dir:
        return True
    path = os.path.join(slot_dir, f"{idx}.lock")
    if not os.path.exists(path):
        return False
    try:
        pid = int(open(path).read().strip())
    except (ValueError, OSError):
        return False
    if pid == MY_PID:
        return True
    try:
        os.kill(pid, 0)
        return True
    except OSError:
        return False


def move_to_slot(new_idx):
    global slot_index
    if not slot_dir:
        return
    old_path = os.path.join(slot_dir, f"{slot_index}.lock")
    new_path = os.path.join(slot_dir, f"{new_idx}.lock")
    try:
        with open(new_path, "w") as f:
            f.write(str(MY_PID))
        os.remove(old_path)
    except OSError:
        pass
    slot_index = new_idx


def cleanup_slot():
    if not slot_dir:
        return
    path = os.path.join(slot_dir, f"{slot_index}.lock")
    try:
        os.remove(path)
    except OSError:
        pass


# --- Position calculation ---


def calc_origin(screen_geo, slot):
    """Returns (x, y) for the notification window."""
    sx, sy, sw, sh = screen_geo
    stack_offset = slot * (WIN_HEIGHT + MARGIN)

    if position == "top-left":
        x = sx + MARGIN
        y = sy + MARGIN + stack_offset
    elif position == "top-center":
        x = sx + (sw - WIN_WIDTH) // 2
        y = sy + MARGIN + stack_offset
    elif position == "top-right":
        x = sx + sw - WIN_WIDTH - MARGIN
        y = sy + MARGIN + stack_offset
    elif position == "bottom-left":
        x = sx + MARGIN
        y = sy + sh - WIN_HEIGHT - MARGIN - stack_offset
    elif position == "bottom-center":
        x = sx + (sw - WIN_WIDTH) // 2
        y = sy + sh - WIN_HEIGHT - MARGIN - stack_offset
    elif position == "bottom-right":
        x = sx + sw - WIN_WIDTH - MARGIN
        y = sy + sh - WIN_HEIGHT - MARGIN - stack_offset
    elif position == "center":
        x = sx + (sw - WIN_WIDTH) // 2
        half = stack_offset // 2
        y = sx + (sh - WIN_HEIGHT) // 2 + (-half if slot % 2 == 0 else half)
    else:  # default top-right
        x = sx + sw - WIN_WIDTH - MARGIN
        y = sy + MARGIN + stack_offset

    return x, y


# --- CSS ---

CSS = b"""
window {
    background-color: transparent;
}

#notification-box {
    background-color: rgba(30, 30, 30, 0.88);
    border-radius: 16px;
    border: 1px solid rgba(255, 255, 255, 0.12);
    padding: 12px 14px;
}

#sender-label {
    color: rgba(255, 255, 255, 0.92);
    font-weight: bold;
    font-size: 13px;
}

#phrase-label {
    color: rgba(255, 255, 255, 0.50);
    font-size: 12px;
}
"""

# --- Window ---


class NotificationWindow(Gtk.Window):
    def __init__(self):
        super().__init__(type=Gtk.WindowType.POPUP)

        self.set_decorated(False)
        self.set_resizable(False)
        self.set_default_size(WIN_WIDTH, WIN_HEIGHT)
        self.set_keep_above(True)
        self.set_skip_taskbar_hint(True)
        self.set_skip_pager_hint(True)
        self.stick()

        # Transparency
        screen = self.get_screen()
        visual = screen.get_rgba_visual()
        if visual:
            self.set_visual(visual)
        self.set_app_paintable(True)

        # Apply CSS
        css_provider = Gtk.CssProvider()
        css_provider.load_from_data(CSS)
        Gtk.StyleContext.add_provider_for_screen(
            screen, css_provider, Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION
        )

        # Main container
        event_box = Gtk.EventBox()
        event_box.connect("button-press-event", self._on_click)
        self.add(event_box)

        box = Gtk.Box(orientation=Gtk.Orientation.HORIZONTAL, spacing=10)
        box.set_name("notification-box")
        event_box.add(box)

        # Icon
        if icon_path and os.path.exists(icon_path):
            try:
                pixbuf = GdkPixbuf.Pixbuf.new_from_file_at_scale(
                    icon_path, 40, 40, True
                )
                icon = Gtk.Image.new_from_pixbuf(pixbuf)
                box.pack_start(icon, False, False, 0)
            except GLib.Error:
                pass

        # Text
        text_box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=2)
        text_box.set_valign(Gtk.Align.CENTER)
        box.pack_start(text_box, True, True, 0)

        sender_label = Gtk.Label()
        sender_label.set_name("sender-label")
        sender_label.set_markup(
            f"<b>{GLib.markup_escape_text(sender_name)} in {GLib.markup_escape_text(project_name)}</b>"
        )
        sender_label.set_xalign(0)
        sender_label.set_ellipsize(Pango.EllipsizeMode.END)
        text_box.pack_start(sender_label, False, False, 0)

        if phrase:
            phrase_label = Gtk.Label(label=phrase)
            phrase_label.set_name("phrase-label")
            phrase_label.set_xalign(0)
            phrase_label.set_ellipsize(Pango.EllipsizeMode.END)
            text_box.pack_start(phrase_label, False, False, 0)

        self._screen_geo = None
        self._position_window()

        # Reflow timer
        if slot_dir:
            GLib.timeout_add(500, self._check_reflow)

        # Auto-dismiss
        if dismiss_secs > 0:
            GLib.timeout_add(int(dismiss_secs * 1000), self._dismiss)

    def _get_screen_geo(self):
        display = Gdk.Display.get_default()
        monitor = display.get_primary_monitor() or display.get_monitor(0)
        if monitor:
            geo = monitor.get_workarea()
            return (geo.x, geo.y, geo.width, geo.height)
        # Fallback
        screen = self.get_screen()
        return (0, 0, screen.get_width(), screen.get_height())

    def _position_window(self):
        self._screen_geo = self._get_screen_geo()
        x, y = calc_origin(self._screen_geo, slot_index)
        self.move(x, y)

    def _check_reflow(self):
        global slot_index
        for s in range(slot_index):
            if not is_slot_occupied(s):
                move_to_slot(s)
                self._position_window()
                break
        return True  # keep timer running

    def _on_click(self, widget, event):
        # Try to focus parent app using wmctrl or xdotool
        _try_focus_parent()
        self._dismiss()

    def _dismiss(self):
        cleanup_slot()
        Gtk.main_quit()
        return False


def _try_focus_parent():
    """Best-effort focus of the terminal/IDE that spawned us."""
    ppid = os.getppid()
    # Try xdotool first
    os.system(f"xdotool search --pid {ppid} windowactivate 2>/dev/null")


def main():
    win = NotificationWindow()
    win.show_all()
    Gtk.main()


if __name__ == "__main__":
    main()
