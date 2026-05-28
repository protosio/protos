#!/usr/bin/env swift
import AppKit
import ApplicationServices
import Foundation

let bundleID = "io.protos.macos"
let axButtonRole = kAXButtonRole as String
let axCellRole = kAXCellRole as String
let axRowRole = kAXRowRole as String
let axStaticTextRole = kAXStaticTextRole as String
let axTextAreaRole = kAXTextAreaRole as String
let axTextFieldRole = kAXTextFieldRole as String
let axWindowRole = kAXWindowRole as String

struct Node {
    let element: AXUIElement
    let role: String
    let title: String
    let value: String
    let description: String
    let help: String
    let placeholder: String
    let enabled: Bool?
    let frame: CGRect?
    let children: [Node]

    var ownText: String {
        [title, value, description, help, placeholder]
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .joined(separator: " ")
    }

    var text: String {
        ([ownText] + children.map(\.text))
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .joined(separator: " ")
    }
}

enum SmokeError: Error, CustomStringConvertible {
    case notTrusted
    case appNotRunning(String)
    case noWindow
    case noElement(String)
    case noFrame(String)
    case axSetFailed(String, AXError)

    var description: String {
        switch self {
        case .notTrusted:
            return "Accessibility is not trusted for this process."
        case .appNotRunning(let id):
            return "App is not running: \(id)"
        case .noWindow:
            return "Protos has no accessible window."
        case .noElement(let label):
            return "Could not find UI element: \(label)"
        case .noFrame(let label):
            return "UI element has no usable frame: \(label)"
        case .axSetFailed(let label, let error):
            return "Could not set \(label): \(error.rawValue)"
        }
    }
}

func attr<T>(_ element: AXUIElement, _ name: String, as type: T.Type) -> T? {
    var raw: CFTypeRef?
    guard AXUIElementCopyAttributeValue(element, name as CFString, &raw) == .success else {
        return nil
    }
    return raw as? T
}

func stringAttr(_ element: AXUIElement, _ name: String) -> String {
    if let value = attr(element, name, as: String.self) {
        return value
    }
    if let number = attr(element, name, as: NSNumber.self) {
        return number.stringValue
    }
    return ""
}

func boolAttr(_ element: AXUIElement, _ name: String) -> Bool? {
    attr(element, name, as: NSNumber.self)?.boolValue
}

func rectAttr(_ element: AXUIElement, _ name: String) -> CGRect? {
    var raw: CFTypeRef?
    guard AXUIElementCopyAttributeValue(element, name as CFString, &raw) == .success,
          let value = raw,
          CFGetTypeID(value) == AXValueGetTypeID()
    else {
        return nil
    }

    let axValue = value as! AXValue
    var rect = CGRect.zero
    if AXValueGetType(axValue) == .cgRect, AXValueGetValue(axValue, .cgRect, &rect) {
        return rect
    }

    var point = CGPoint.zero
    if AXValueGetType(axValue) == .cgPoint, AXValueGetValue(axValue, .cgPoint, &point) {
        return CGRect(origin: point, size: .zero)
    }

    var size = CGSize.zero
    if AXValueGetType(axValue) == .cgSize, AXValueGetValue(axValue, .cgSize, &size) {
        return CGRect(origin: .zero, size: size)
    }

    return nil
}

func frameAttr(_ element: AXUIElement) -> CGRect? {
    if let frame = rectAttr(element, "AXFrame") {
        return frame
    }
    guard let position = rectAttr(element, kAXPositionAttribute as String),
          let size = rectAttr(element, kAXSizeAttribute as String)
    else {
        return nil
    }
    return CGRect(origin: position.origin, size: size.size)
}

func childElements(_ element: AXUIElement) -> [AXUIElement] {
    var seen = Set<UInt>()
    var elements: [AXUIElement] = []

    func append(_ candidate: AXUIElement?) {
        guard let candidate else { return }
        let id = CFHash(candidate)
        guard !seen.contains(id) else { return }
        seen.insert(id)
        elements.append(candidate)
    }

    for child in attr(element, kAXChildrenAttribute as String, as: [AXUIElement].self) ?? [] {
        append(child)
    }
    for window in attr(element, kAXWindowsAttribute as String, as: [AXUIElement].self) ?? [] {
        append(window)
    }
    append(attr(element, kAXMainWindowAttribute as String, as: AXUIElement.self))
    return elements
}

func buildNode(_ element: AXUIElement, depth: Int, maxDepth: Int) -> Node {
    let role = stringAttr(element, kAXRoleAttribute as String)
    let children = depth < maxDepth
        ? childElements(element).map { buildNode($0, depth: depth + 1, maxDepth: maxDepth) }
        : []
    return Node(
        element: element,
        role: role,
        title: stringAttr(element, kAXTitleAttribute as String),
        value: stringAttr(element, kAXValueAttribute as String),
        description: stringAttr(element, kAXDescriptionAttribute as String),
        help: stringAttr(element, kAXHelpAttribute as String),
        placeholder: stringAttr(element, kAXPlaceholderValueAttribute as String),
        enabled: boolAttr(element, kAXEnabledAttribute as String),
        frame: frameAttr(element),
        children: children
    )
}

func runningApp() throws -> NSRunningApplication {
    guard let app = NSRunningApplication.runningApplications(withBundleIdentifier: bundleID).first else {
        throw SmokeError.appNotRunning(bundleID)
    }
    app.activate()
    Thread.sleep(forTimeInterval: 0.25)
    return app
}

func appRoot() throws -> Node {
    guard AXIsProcessTrusted() else {
        throw SmokeError.notTrusted
    }
    let app = try runningApp()
    let element = AXUIElementCreateApplication(app.processIdentifier)
    return buildNode(element, depth: 0, maxDepth: 14)
}

func windowRoot() throws -> Node {
    let root = try appRoot()
    guard let window = flatten(root).first(where: { $0.role == axWindowRole }) else {
        throw SmokeError.noWindow
    }
    return window
}

func flatten(_ node: Node) -> [Node] {
    [node] + node.children.flatMap(flatten)
}

func score(_ node: Node, label: String) -> Int {
    let text = node.text.trimmingCharacters(in: .whitespacesAndNewlines)
    let own = node.ownText.trimmingCharacters(in: .whitespacesAndNewlines)
    var result = 0
    if own == label { result += 1_000 }
    if text == label { result += 700 }
    if own.localizedCaseInsensitiveContains(label) { result += 300 }
    if text.localizedCaseInsensitiveContains(label) { result += 100 }
    switch node.role {
    case axButtonRole:
        result += 80
    case axRowRole, axCellRole:
        result += 55
    case axTextFieldRole:
        result += 40
    case axStaticTextRole:
        result += 10
    default:
        result -= 20
    }
    if node.enabled == false { result -= 500 }
    if let area = node.frame.map({ $0.width * $0.height }), area > 0 {
        result -= min(Int(area / 100_000), 50)
    }
    return result
}

func findNode(label: String, roles: Set<String> = [], predicate: (Node) -> Bool = { _ in true }) throws -> Node {
    let nodes = flatten(try windowRoot())
    let candidates = nodes
        .filter { roles.isEmpty || roles.contains($0.role) }
        .filter(predicate)
        .map { (node: $0, score: score($0, label: label)) }
        .filter { $0.score > 0 }
        .sorted {
            if $0.score == $1.score {
                let leftArea = $0.node.frame.map { $0.width * $0.height } ?? .greatestFiniteMagnitude
                let rightArea = $1.node.frame.map { $0.width * $0.height } ?? .greatestFiniteMagnitude
                return leftArea < rightArea
            }
            return $0.score > $1.score
        }
    guard let best = candidates.first?.node else {
        throw SmokeError.noElement(label)
    }
    return best
}

func detailMinX() throws -> CGFloat {
    let nodes = flatten(try windowRoot())
    if let splitter = nodes.first(where: { $0.role == kAXSplitterRole as String })?.frame {
        return splitter.maxX
    }
    if let window = nodes.first?.frame {
        return window.minX + 160
    }
    return 0
}

func click(_ node: Node, label: String) throws {
    guard let frame = node.frame, frame.width > 0, frame.height > 0 else {
        throw SmokeError.noFrame(label)
    }
    let point = CGPoint(x: frame.midX, y: frame.midY)
    let source = CGEventSource(stateID: .hidSystemState)
    let down = CGEvent(mouseEventSource: source, mouseType: .leftMouseDown, mouseCursorPosition: point, mouseButton: .left)
    let up = CGEvent(mouseEventSource: source, mouseType: .leftMouseUp, mouseCursorPosition: point, mouseButton: .left)
    down?.post(tap: .cghidEventTap)
    Thread.sleep(forTimeInterval: 0.05)
    up?.post(tap: .cghidEventTap)
    Thread.sleep(forTimeInterval: 0.45)
}

func setValue(_ value: String, in node: Node, label: String) throws {
    let focusError = AXUIElementSetAttributeValue(node.element, kAXFocusedAttribute as CFString, kCFBooleanTrue)
    if focusError != .success && focusError != .attributeUnsupported {
        throw SmokeError.axSetFailed("focus for \(label)", focusError)
    }
    let setError = AXUIElementSetAttributeValue(node.element, kAXValueAttribute as CFString, value as CFString)
    guard setError == .success else {
        throw SmokeError.axSetFailed("value for \(label)", setError)
    }
    Thread.sleep(forTimeInterval: 0.2)
}

func printTree(_ node: Node, depth: Int = 0) {
    let indent = String(repeating: "  ", count: depth)
    let frame = node.frame.map { " [\(Int($0.minX)),\(Int($0.minY)) \(Int($0.width))x\(Int($0.height))]" } ?? ""
    let enabled = node.enabled.map { $0 ? " enabled" : " disabled" } ?? ""
    let text = node.ownText.isEmpty ? "" : " \"\(node.ownText)\""
    print("\(indent)\(node.role)\(enabled)\(frame)\(text)")
    for child in node.children {
        printTree(child, depth: depth + 1)
    }
}

func assertVisible(_ label: String) throws {
    _ = try findNode(label: label)
}

func assertDetailVisible(_ label: String) throws {
    let minX = try detailMinX()
    _ = try findNode(label: label) { node in
        guard let frame = node.frame else { return false }
        return frame.minX >= minX
    }
}

func assertDetailHeader(_ label: String) throws {
    let minX = try detailMinX()
    _ = try findNode(label: label, roles: [axStaticTextRole]) { node in
        guard let frame = node.frame else { return false }
        return frame.minX >= minX && node.ownText == label
    }
}

func detailContains(_ label: String) -> Bool {
    let minX = (try? detailMinX()) ?? 0
    let nodes = (try? flatten(windowRoot())) ?? []
    return nodes.contains { node in
        guard let frame = node.frame, frame.minX >= minX else { return false }
        return node.text.localizedCaseInsensitiveContains(label)
    }
}

func clickLabel(_ label: String) throws {
    let node = try findNode(
        label: label,
        roles: [axButtonRole, axRowRole, axCellRole, axStaticTextRole]
    )
    try click(node, label: label)
}

func setField(_ placeholder: String, _ value: String) throws {
    let node = try findNode(label: placeholder, roles: [axTextFieldRole, axTextAreaRole])
    try setValue(value, in: node, label: placeholder)
}

let args = Array(CommandLine.arguments.dropFirst())

do {
    guard let command = args.first else {
        print("usage: protos-ui-smoke.swift dump|click <label>|set <placeholder> <value>|assert <label>|walk-sidebar|network-negative|network-route [instance]")
        exit(2)
    }

    switch command {
    case "dump":
        printTree(try windowRoot())
    case "click":
        guard args.count >= 2 else { exit(2) }
        try clickLabel(args[1])
        print("clicked \(args[1])")
    case "set":
        guard args.count >= 3 else { exit(2) }
        try setField(args[1], args[2])
        print("set \(args[1])")
    case "assert":
        guard args.count >= 2 else { exit(2) }
        try assertVisible(args[1])
        print("visible \(args[1])")
    case "walk-sidebar":
        let sections = [
            ("Overview", "Initialize"),
            ("Apps", "No apps"),
            ("Provisioners", "Supported"),
            ("Instances", "Machine"),
            ("Network", "Gateway"),
            ("Releases", "Available Releases"),
            ("P2P DVC", "Commits"),
        ]
        for (section, detailMarker) in sections {
            try clickLabel(section)
            try assertDetailHeader(section)
            try assertDetailVisible(detailMarker)
            print("section \(section) ok")
        }
    case "network-negative":
        try clickLabel("Network")
        try assertVisible("Exit Routes")
        try setField("Target instance", "does-not-exist")
        try clickLabel("Load")
        try assertVisible("does-not-exist")
        print("network negative load surfaced error")
    case "network-route":
        let instance = args.count >= 2 ? args[1] : "usa-exit-ash"
        try clickLabel("Network")
        try assertDetailVisible("Gateway")
        try setField("Instance", instance)
        try setField("CIDRs", "203.0.113.0/24")
        try clickLabel("Set")
        Thread.sleep(forTimeInterval: 2.0)
        if detailContains("Routing local traffic through") {
            print("route set ok")
            try clickLabel("Clear")
            Thread.sleep(forTimeInterval: 1.0)
            try assertDetailVisible("Exit route disabled")
            print("route clear ok")
        } else if detailContains(instance) || detailContains("failed") || detailContains("rpc error") {
            throw SmokeError.noElement("route success output")
        } else {
            throw SmokeError.noElement("route result")
        }
    default:
        eprint("unknown command: \(command)")
        exit(2)
    }
} catch {
    eprint("ERROR: \(error)")
    exit(1)
}

func eprint(_ message: String) {
    if let data = (message + "\n").data(using: .utf8) {
        FileHandle.standardError.write(data)
    }
}
