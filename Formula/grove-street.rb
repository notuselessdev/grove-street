class GroveStreet < Formula
  desc "GTA San Andreas CJ voice notifications for AI coding agents"
  homepage "https://github.com/notuselessdev/grove-street"
  version "0.1.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_arm64.tar.gz"
      sha256 "7dc4eeae357069a21e5c08a23d8f91db901cf8d30d4727a9bc1b8cbeeee54463"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_darwin_amd64.tar.gz"
      sha256 "a357ce9366b0b9c8c40d48165ef101fcd3f2a730a9516962b6dec33a4b79520f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_arm64.tar.gz"
      sha256 "dd2148eaaad097d0f15d8bb61eb39dc878273524ffaa6f5110fca63580ce2417"
    else
      url "https://github.com/notuselessdev/grove-street/releases/download/v#{version}/grove-street_linux_amd64.tar.gz"
      sha256 "15e5ed04c2c9e7fbefb8fe3591e3fede4e5a824c683567f89c154844036681a0"
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
