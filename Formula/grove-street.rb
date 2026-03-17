class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.3.9"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "2cc9fe697d2932c528c93039775995134b0b4b5bd700f87a7ec5fb9bb82195ab"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "156fb0147448d26a1b4f3429d34c413bec52c07782edec5ee078cd3e4288db9a"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "035ccf02dd211ad12bf897ab586406fa9226f82195cf26ede8cb12a2bdbf53b7"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "21dc2de6c694ab85a175765ed16661e9787e57a6a9ef0e09a2e5ae2b3e13d389"
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
