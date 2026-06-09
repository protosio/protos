import Cocoa
import FlutterMacOS

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)
    ProtosStatusBarController.shared.install()

    super.awakeFromNib()
  }

  override func close() {
    // Keep the embedded core and Flutter view alive when the user closes the window.
    orderOut(nil)
  }
}
