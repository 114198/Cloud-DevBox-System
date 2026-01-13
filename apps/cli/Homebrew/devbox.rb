# Homebrew formula for DevBox CLI
# Install: brew install cloud-devbox/tap/devbox

class Devbox < Formula
  desc "Cloud DevBox CLI - Manage your cloud development environments"
  homepage "https://github.com/your-org/cloud-devbox"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/your-org/cloud-devbox/releases/download/v#{version}/devbox-#{version}-x86_64-apple-darwin.tar.gz"
      sha256 "PLACEHOLDER_SHA256_MACOS_X64"
    end

    on_arm do
      url "https://github.com/your-org/cloud-devbox/releases/download/v#{version}/devbox-#{version}-aarch64-apple-darwin.tar.gz"
      sha256 "PLACEHOLDER_SHA256_MACOS_ARM64"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/your-org/cloud-devbox/releases/download/v#{version}/devbox-#{version}-x86_64-unknown-linux-gnu.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_X64"
    end

    on_arm do
      url "https://github.com/your-org/cloud-devbox/releases/download/v#{version}/devbox-#{version}-aarch64-unknown-linux-gnu.tar.gz"
      sha256 "PLACEHOLDER_SHA256_LINUX_ARM64"
    end
  end

  def install
    bin.install "devbox"
    
    # Generate shell completions
    generate_completions_from_executable(bin/"devbox", "completions")
  end

  test do
    assert_match "devbox #{version}", shell_output("#{bin}/devbox --version")
  end
end
