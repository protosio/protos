import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(expandedInitialFrame(from: windowFrame), display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)
    ProtosStatusBarController.shared.install()

    super.awakeFromNib()
  }

  override func close() {
    // Keep the embedded core and Flutter view alive when the user closes the window.
    orderOut(nil)
  }

  private func expandedInitialFrame(from frame: NSRect) -> NSRect {
    let targetSize = NSSize(width: frame.width * 1.5, height: frame.height * 1.5)
    guard let visibleFrame = self.screen?.visibleFrame ?? NSScreen.main?.visibleFrame else {
      return NSRect(
        x: frame.midX - targetSize.width / 2,
        y: frame.midY - targetSize.height / 2,
        width: targetSize.width,
        height: targetSize.height
      )
    }
    let finalSize = NSSize(
      width: min(targetSize.width, visibleFrame.width),
      height: min(targetSize.height, visibleFrame.height)
    )
    return NSRect(
      x: visibleFrame.midX - finalSize.width / 2,
      y: visibleFrame.midY - finalSize.height / 2,
      width: finalSize.width,
      height: finalSize.height
    )
  }
}
