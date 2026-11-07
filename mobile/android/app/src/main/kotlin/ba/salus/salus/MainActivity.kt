package ba.salus.salus

import android.graphics.Bitmap
import android.graphics.ImageDecoder
import android.net.Uri
import android.os.Build
import androidx.media3.common.MediaItem
import androidx.media3.common.MimeTypes
import androidx.media3.transformer.Composition
import androidx.media3.transformer.ExportException
import androidx.media3.transformer.ExportResult
import androidx.media3.transformer.Transformer
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.util.concurrent.Executors

// The conversions of docs/attachments/SPEC.md §2 "Mobile" for lib/files.dart: a HEIC or HEIF photo to JPEG, and a
// video that is not H.264/AAC MP4 to one (Media3 Transformer), so that everything plays in Chrome. The result goes
// to the app's cache directory; the Dart side strips its metadata (lib/file_prep.dart) and deletes it once
// uploaded. Errors carry no message, so no path or file detail reaches Dart. Nothing here logs.
class MainActivity : FlutterActivity() {
    private val io = Executors.newSingleThreadExecutor()

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, "ba.salus.salus/files").setMethodCallHandler { call, result ->
            val path = call.argument<String>("path")
            when {
                call.method != "convert" || path == null -> result.notImplemented()
                call.argument<String>("kind") == "photo" -> io.execute {
                    val out = runCatching { photoToJpeg(File(path)) }.getOrNull()
                    runOnUiThread { if (out != null) result.success(out.path) else result.error("convert", null, null) }
                }
                else -> videoToMp4(File(path), result)
            }
        }
    }

    // Decoded upright (ImageDecoder applies the photo's rotation), written as a JPEG with no metadata at all.
    private fun photoToJpeg(src: File): File {
        check(Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) { "HEIF needs Android 9" }
        val bitmap = ImageDecoder.decodeBitmap(ImageDecoder.createSource(src)) { decoder, _, _ ->
            decoder.allocator = ImageDecoder.ALLOCATOR_SOFTWARE
        }
        val out = File.createTempFile("salus-converted-", ".jpg", cacheDir)
        try {
            out.outputStream().use { check(bitmap.compress(Bitmap.CompressFormat.JPEG, 90, it)) { "compress" } }
        } catch (e: Exception) {
            out.delete()
            throw e
        } finally {
            bitmap.recycle()
        }
        return out
    }

    // Transformer runs on the main thread's looper and reports there.
    private fun videoToMp4(src: File, result: MethodChannel.Result) {
        val out = File.createTempFile("salus-converted-", ".mp4", cacheDir)
        val transformer = Transformer.Builder(this)
            .setVideoMimeType(MimeTypes.VIDEO_H264)
            .setAudioMimeType(MimeTypes.AUDIO_AAC)
            .addListener(object : Transformer.Listener {
                override fun onCompleted(composition: Composition, exportResult: ExportResult) {
                    result.success(out.path)
                }

                override fun onError(composition: Composition, exportResult: ExportResult, exportException: ExportException) {
                    out.delete()
                    result.error("convert", null, null)
                }
            })
            .build()
        transformer.start(MediaItem.fromUri(Uri.fromFile(src)), out.path)
    }
}
