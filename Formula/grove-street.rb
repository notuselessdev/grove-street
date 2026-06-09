class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.5.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "3f0ce8d802bcd12bf5ad8c0029132b7a466781307fa6f666543e68faf8087625"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "76a5620731bea17ddb25a6668ef140592ffcfd1d4da43c77f5a2b937e3009061"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "cc359085a0593db1e0fe7a4c00061ea4f967ccbe88449c5d91c75ed39acde43b"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "854a68e74d925eef3c1052fa3bfbee3460302c68ddd2a393a26b650152fee775"
    end
  end

  def install
    bin.install "grove-street"
    (share/"grove-street").install "icon.png" if File.exist?("icon.png")
    (share/"grove-street").install "grove-notify.swift" if File.exist?("grove-notify.swift")
    (share/"grove-street").install "grove-notify.py" if File.exist?("grove-notify.py")
    (share/"grove-street").install "grove-notify.ps1" if File.exist?("grove-notify.ps1")
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

      🎮 Enjoying Grove Street? Help CJ spread the word!

        ⭐ Give us a star: https://github.com/notuselessdev/grove-street
        🌐 Website:        https://notuseless.dev
        👤 Follow the dev: https://x.com/notuselessdev

      "Respect is everything, CJ." 🫡
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
