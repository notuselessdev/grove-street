class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.3.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "33fe0043c09fdadcb6ce4d08d296a0896e6c051cb5fe2bd12809eaa2261fa429"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "f319b49fb9acbb3096fd308e8205edb43d7d5740acaa611f3a37e6bf3ebd5783"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "e4a16e4a198c485522e7d2474effc4097dd0815b27fe06c8db742856a697ac94"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "244f3620a8c864ff9a493185bf13d3d92ce2158f79b7b1df4e7006a9333c23a3"
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
