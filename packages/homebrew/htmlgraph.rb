class Htmlgraph < Formula
  desc "Local-first observability and coordination platform for AI-assisted development"
  homepage "https://github.com/shakestzd/htmlgraph"
  version "0.35.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/shakestzd/htmlgraph/releases/download/go/v#{version}/htmlgraph_#{version}_darwin_arm64.tar.gz"
      sha256 "SHA256_DARWIN_ARM64"
    else
      url "https://github.com/shakestzd/htmlgraph/releases/download/go/v#{version}/htmlgraph_#{version}_darwin_amd64.tar.gz"
      sha256 "SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/shakestzd/htmlgraph/releases/download/go/v#{version}/htmlgraph_#{version}_linux_arm64.tar.gz"
      sha256 "SHA256_LINUX_ARM64"
    else
      url "https://github.com/shakestzd/htmlgraph/releases/download/go/v#{version}/htmlgraph_#{version}_linux_amd64.tar.gz"
      sha256 "SHA256_LINUX_AMD64"
    end
  end

  def install
    bin.install "htmlgraph"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/htmlgraph version")
  end
end
