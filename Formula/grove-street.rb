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

  depends_on "terminal-notifier" => :recommended

  def install
    bin.install "grove-street"
    (share/"grove-street").install "icon.png" if File.exist?("icon.png")
  end

  def post_install
    system bin/"grove-street", "setup"
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
end
