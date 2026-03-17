class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "101e13fe118b83bef8df240f7032fdfb6f7c9ded2ae7fb225e8d3a39e8632313"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "6ea8c6ca37e71d2f69a4fa66d9080c124fe3c3bbf2e6a40de1bc863899d4db6f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "3e3dcfe76fd6c1aca015b48fd24b8c5b247fd0c26702138d121464287cf825aa"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "297f23d955f99cbf110a550c4484ec44405ab6c675e68e756cc51d72e99d0fbf"
    end
  end

  def install
    bin.install "grove-street"
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
