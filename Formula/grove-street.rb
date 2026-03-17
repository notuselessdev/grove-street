class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.2.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "0f7b19b05a14f59294689f03cbf5d82cbc56f46e095d7e051c95d4cec037b625"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "891eea35c20db475df442104ae6cfff9615e00fb9b55501e030eba4a23a675d9"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "063b7c9d1f59cf24b4d965d04ed76ef950c6bc35275b0550f25133c79b243132"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "4d507b892eabb8494b356767a263f3ba71157df144f25f4f08cfb325d3f3ad73"
    end
  end

  def install
    bin.install "grove-street"
    (share/"grove-street").install "icon.png" if File.exist?("icon.png")
    (share/"grove-street").install "mac-overlay.js" if File.exist?("mac-overlay.js")
  end

  def post_install
    system bin/"grove-street", "setup"
  end

  def post_uninstall
    remove_hooks_from_json(File.join(Dir.home, ".claude", "settings.json"))
    remove_hooks_from_json(File.join(Dir.home, ".cursor", "hooks.json"))
    remove_hooks_from_json(File.join(Dir.home, ".codeium", "windsurf", "hooks.json"))
    remove_hooks_from_json(File.join(Dir.home, ".github", "hooks", "hooks.json"))
    kiro = File.join(Dir.home, ".kiro", "agents", "grove-street.json")
    File.delete(kiro) if File.exist?(kiro)
    grove_dir = File.join(Dir.home, ".grove-street")
    ohai "Removed grove-street hooks from all IDEs"
    ohai "To remove sounds and config: rm -rf #{grove_dir}"
  end

  def caveats
    <<~EOS
      Grove Street. Home. CJ is watching your terminal now.

      Run 'grove-street setup' to register hooks in Claude Code.
      Add your CJ voice lines to: ~/.grove-street/sounds/<category>/

      Categories: session_start, task_complete, task_error,
                  input_required, resource_limit, user_spam
    EOS
  end

  test do
    assert_match "grove-street", shell_output("#{bin}/grove-street version")
  end

  private

  def remove_hooks_from_json(path)
    return unless File.exist?(path)

    require "json"
    settings = JSON.parse(File.read(path))
    hooks = settings["hooks"]
    return unless hooks.is_a?(Hash)

    modified = false
    hooks.each do |event, entries|
      next unless entries.is_a?(Array)

      filtered = entries.reject do |entry|
        next false unless entry.is_a?(Hash)

        cmd = entry.dig("command").to_s
        nested = Array(entry["hooks"]).any? { |h| h.is_a?(Hash) && h["command"].to_s.include?("grove-street") }
        hit = cmd.include?("grove-street") || nested
        modified = true if hit
        hit
      end

      if filtered.empty?
        hooks.delete(event)
      else
        hooks[event] = filtered
      end
    end

    return unless modified

    settings.delete("hooks") if hooks.empty?
    File.write(path, JSON.pretty_generate(settings) + "\n")
  end
end
