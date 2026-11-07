import AVFoundation
import Flutter
import ImageIO
import UIKit
import UniformTypeIdentifiers

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    if let registrar = engineBridge.pluginRegistry.registrar(forPlugin: "SalusFiles") {
      Conversion.register(with: registrar.messenger())
    }
  }
}

// The conversions of docs/attachments/SPEC.md §2 "Mobile" for lib/files.dart: a HEIC or HEIF photo to JPEG, and a
// video that is not H.264/AAC MP4 to one (AVAssetExportSession), so that everything plays in Chrome. The result goes
// to the temporary directory; the Dart side strips its metadata (lib/file_prep.dart) and deletes it once uploaded.
// Errors carry no message, so no path or file detail reaches Dart. Nothing here logs.
enum Conversion {
  static func register(with messenger: FlutterBinaryMessenger) {
    let channel = FlutterMethodChannel(name: "ba.salus.salus/files", binaryMessenger: messenger)
    channel.setMethodCallHandler { call, result in
      guard call.method == "convert", let args = call.arguments as? [String: Any],
        let path = args["path"] as? String, let kind = args["kind"] as? String
      else {
        result(FlutterMethodNotImplemented)
        return
      }
      let done: (URL?) -> Void = { out in
        DispatchQueue.main.async { result(out?.path ?? FlutterError(code: "convert", message: nil, details: nil)) }
      }
      let src = URL(fileURLWithPath: path)
      if kind == "photo" {
        DispatchQueue.global(qos: .userInitiated).async { done(photoToJpeg(src)) }
      } else {
        videoToMp4(src, done: done)
      }
    }
  }

  private static func temporary(_ ext: String) -> URL {
    FileManager.default.temporaryDirectory.appendingPathComponent("salus-converted-\(UUID().uuidString).\(ext)")
  }

  // The pixels and the orientation only: no location, device, time or other metadata goes with them.
  static func photoToJpeg(_ src: URL) -> URL? {
    guard let source = CGImageSourceCreateWithURL(src as CFURL, nil),
      let image = CGImageSourceCreateImageAtIndex(source, 0, nil)
    else { return nil }
    let properties = CGImageSourceCopyPropertiesAtIndex(source, 0, nil) as? [CFString: Any]
    let orientation = properties?[kCGImagePropertyOrientation] ?? 1
    let out = temporary("jpg")
    guard let destination = CGImageDestinationCreateWithURL(out as CFURL, UTType.jpeg.identifier as CFString, 1, nil)
    else { return nil }
    let options = [kCGImagePropertyOrientation: orientation, kCGImageDestinationLossyCompressionQuality: 0.9]
    CGImageDestinationAddImage(destination, image, options as CFDictionary)
    guard CGImageDestinationFinalize(destination) else {
      try? FileManager.default.removeItem(at: out)
      return nil
    }
    return out
  }

  // The 1920x1080 preset writes H.264 video and AAC sound; the metadata is dropped and the rest filtered for sharing.
  static func videoToMp4(_ src: URL, done: @escaping (URL?) -> Void) {
    guard let export = AVAssetExportSession(asset: AVURLAsset(url: src), presetName: AVAssetExportPreset1920x1080)
    else {
      done(nil)
      return
    }
    let out = temporary("mp4")
    export.outputURL = out
    export.outputFileType = .mp4
    export.shouldOptimizeForNetworkUse = true
    export.metadata = []
    export.metadataItemFilter = .forSharing()
    export.exportAsynchronously {
      if export.status == .completed {
        done(out)
      } else {
        try? FileManager.default.removeItem(at: out)
        done(nil)
      }
    }
  }
}
