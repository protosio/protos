import Cocoa
import FlutterMacOS

@main
class AppDelegate: FlutterAppDelegate {
  override func applicationDidFinishLaunching(_ notification: Notification) {
    super.applicationDidFinishLaunching(notification)
    ProtosStatusBarController.shared.install()
  }

  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return false
  }

  override func applicationShouldHandleReopen(
    _ sender: NSApplication,
    hasVisibleWindows flag: Bool
  ) -> Bool {
    ProtosStatusBarController.shared.showMainWindow()
    return true
  }

  override func applicationSupportsSecureRestorableState(_ app: NSApplication) -> Bool {
    return true
  }
}

final class ProtosStatusBarController: NSObject {
  static let shared = ProtosStatusBarController()

  private var statusItem: NSStatusItem?

  func install() {
    if statusItem != nil {
      return
    }

    let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.squareLength)
    statusItem = item

    if let button = item.button {
      button.image = makeStatusIcon()
      button.imagePosition = .imageOnly
      button.toolTip = "Protos"
    }

    let menu = NSMenu()
    menu.addItem(menuItem("Show Protos", action: #selector(showProtos(_:))))
    menu.addItem(menuItem("Hide Protos", action: #selector(hideProtos(_:))))
    menu.addItem(.separator())
    menu.addItem(menuItem("Quit Protos", action: #selector(quitProtos(_:)), keyEquivalent: "q"))
    item.menu = menu
  }

  private func menuItem(
    _ title: String,
    action: Selector,
    keyEquivalent: String = ""
  ) -> NSMenuItem {
    let item = NSMenuItem(title: title, action: action, keyEquivalent: keyEquivalent)
    item.target = self
    return item
  }

  private func makeStatusIcon() -> NSImage {
    if #available(macOS 11.0, *) {
      let image = NSImage(
        systemSymbolName: "bolt.horizontal.circle.fill",
        accessibilityDescription: "Protos"
      )
      image?.isTemplate = true
      return image ?? NSImage()
    }

    let image = NSImage(size: NSSize(width: 18, height: 18))
    image.lockFocus()

    let attributes: [NSAttributedString.Key: Any] = [
      .font: NSFont.systemFont(ofSize: 13, weight: .semibold),
      .foregroundColor: NSColor.black,
      .paragraphStyle: centeredParagraphStyle(),
    ]
    "P".draw(in: NSRect(x: 0, y: 1, width: 18, height: 16), withAttributes: attributes)

    image.unlockFocus()
    image.isTemplate = true
    return image
  }

  private func centeredParagraphStyle() -> NSParagraphStyle {
    let style = NSMutableParagraphStyle()
    style.alignment = .center
    return style
  }

  @objc private func showProtos(_ sender: Any?) {
    showMainWindow()
  }

  @objc private func hideProtos(_ sender: Any?) {
    mainFlutterWindow()?.orderOut(sender)
  }

  @objc private func quitProtos(_ sender: Any?) {
    NSApp.terminate(sender)
  }

  func showMainWindow() {
    guard let window = mainFlutterWindow() else {
      return
    }
    window.makeKeyAndOrderFront(nil)
    NSApp.activate(ignoringOtherApps: true)
  }

  private func mainFlutterWindow() -> NSWindow? {
    return NSApp.windows.first { $0 is MainFlutterWindow } ?? NSApp.mainWindow
  }
}
