package app.rooms.ui

import android.app.Presentation
import android.content.ActivityNotFoundException
import android.content.Context
import android.content.Intent
import android.graphics.Color as AndroidColor
import android.hardware.display.DisplayManager
import android.os.Bundle
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import android.view.Display
import android.view.Gravity
import android.view.View
import android.view.ViewGroup.LayoutParams.WRAP_CONTENT
import android.widget.FrameLayout
import android.widget.TextView
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Cast
import androidx.compose.material.icons.outlined.CastConnected
import androidx.compose.material.icons.outlined.Link
import androidx.compose.material.icons.outlined.Tv
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.SideEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import app.rooms.neko.WebRtc
import org.webrtc.RendererCommon
import org.webrtc.SurfaceViewRenderer
import org.webrtc.VideoTrack

// rememberTvDisplay is a second screen the video can be shown on: a TV over
// HDMI or USB-C, or a wireless one (Miracast, Smart View, some Cast setups)
@Composable
fun rememberTvDisplay(): Display? {
    val context = LocalContext.current
    val manager = remember(context) { context.getSystemService(DisplayManager::class.java) }
    fun find() = manager.getDisplays(DisplayManager.DISPLAY_CATEGORY_PRESENTATION).firstOrNull()
    var display by remember { mutableStateOf(find()) }
    DisposableEffect(manager) {
        val listener = object : DisplayManager.DisplayListener {
            // the same screen keeps its Display, so it is not shown anew
            private fun update() {
                val d = find()
                if (d?.displayId != display?.displayId) display = d
            }
            override fun onDisplayAdded(displayId: Int) = update()
            override fun onDisplayRemoved(displayId: Int) = update()
            override fun onDisplayChanged(displayId: Int) = update()
        }
        manager.registerDisplayListener(listener, Handler(Looper.getMainLooper()))
        onDispose { manager.unregisterDisplayListener(listener) }
    }
    return display
}

// TvVideo shows the room video full screen on the TV while it is composed,
// the phone keeps its own picture to control the browser and chat
@Composable
fun TvVideo(display: Display, track: VideoTrack?) {
    val context = LocalContext.current
    val presentation = remember(display.displayId) { TvPresentation(context, display) }
    DisposableEffect(presentation) {
        // the TV can be gone again by now
        runCatching { presentation.show() }
        onDispose { presentation.dismiss() }
    }
    SideEffect { presentation.setTrack(track) }
}

private class TvPresentation(owner: Context, display: Display) : Presentation(owner, display) {
    private var renderer: SurfaceViewRenderer? = null
    private var waiting: TextView? = null
    private var track: VideoTrack? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // wrapped in the middle of the screen, so the picture keeps its shape
        // with black bars instead of being stretched or cut
        val r = SurfaceViewRenderer(context).apply {
            init(WebRtc.egl.eglBaseContext, null)
            setScalingType(RendererCommon.ScalingType.SCALE_ASPECT_FIT)
            setEnableHardwareScaler(true)
        }
        val w = TextView(context).apply {
            text = "Connecting…"
            setTextColor(AndroidColor.WHITE)
            textSize = 24f
        }
        setContentView(FrameLayout(context).apply {
            setBackgroundColor(AndroidColor.BLACK)
            addView(r, FrameLayout.LayoutParams(WRAP_CONTENT, WRAP_CONTENT, Gravity.CENTER))
            addView(w, FrameLayout.LayoutParams(WRAP_CONTENT, WRAP_CONTENT, Gravity.CENTER))
        })
        renderer = r
        waiting = w
        attach(track)
    }

    override fun onStop() {
        renderer?.let { r ->
            track?.let { runCatching { it.removeSink(r) } }
            r.release()
        }
        renderer = null
        waiting = null
        super.onStop()
    }

    fun setTrack(t: VideoTrack?) {
        if (t === track) return
        renderer?.let { r -> track?.let { runCatching { it.removeSink(r) } } }
        track = t
        attach(t)
    }

    private fun attach(t: VideoTrack?) {
        val r = renderer ?: return
        t?.let { runCatching { it.addSink(r) } }
        waiting?.visibility = if (t == null) View.VISIBLE else View.GONE
    }
}

@Composable
fun CastButton(tv: Display?, showing: Boolean, tint: Color, onClick: () -> Unit) {
    val on = tv != null && showing
    IconButton(onClick = onClick) {
        Icon(
            if (on) Icons.Outlined.CastConnected else Icons.Outlined.Cast,
            if (on) "Showing on ${tv?.name}" else "Watch on a TV",
            tint = if (on) MaterialTheme.colorScheme.primary else tint,
        )
    }
}

// CastDialog is how a room gets onto a TV: a screen that is already
// connected shows the video by itself, otherwise the phone connects to one,
// or the room link goes to a device with its own browser
@Composable
fun CastDialog(tv: Display?, showing: Boolean, onShowing: (Boolean) -> Unit, link: String, onDismiss: () -> Unit) {
    val context = LocalContext.current
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text(if (tv != null) "Watching on ${tv.name}" else "Watch on a TV") },
        text = {
            Column {
                if (tv != null) {
                    Text(
                        if (showing) "The video fills the TV, your phone stays the remote and the chat."
                        else "The TV is connected but shows your phone's screen.",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                } else {
                    CastOption(
                        Icons.Outlined.Tv,
                        "Cast screen",
                        "Connect to a Chromecast or a TV with screen mirroring (Smart View, Miracast). If the whole phone shows up, turn it sideways for full screen.",
                    ) {
                        onDismiss()
                        openCastSettings(context)
                    }
                    if (link.isNotEmpty()) {
                        CastOption(
                            Icons.Outlined.Link,
                            "Open on another screen",
                            "Send the room link to a smart TV, Android TV or computer and open it in its browser.",
                        ) {
                            onDismiss()
                            context.startActivity(
                                Intent.createChooser(
                                    Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, link),
                                    "Open room on",
                                )
                            )
                        }
                    }
                    Text(
                        "A USB-C to HDMI cable works too: the video moves to the TV by itself.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }
            }
        },
        confirmButton = {
            if (tv != null) {
                TextButton(onClick = { onShowing(!showing); onDismiss() }) {
                    Text(if (showing) "Mirror phone instead" else "Show video on TV")
                }
            } else {
                TextButton(onClick = onDismiss) { Text("Close") }
            }
        },
        dismissButton = {
            if (tv != null) TextButton(onClick = { onDismiss(); openCastSettings(context) }) { Text("Disconnect") }
        },
    )
}

@Composable
private fun CastOption(icon: ImageVector, title: String, text: String, onClick: () -> Unit) {
    Row(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(12.dp))
            .clickable(onClick = onClick)
            .padding(vertical = 10.dp, horizontal = 4.dp),
        verticalAlignment = Alignment.Top,
    ) {
        Icon(icon, null, Modifier.size(24.dp), tint = MaterialTheme.colorScheme.primary)
        Spacer(Modifier.width(14.dp))
        Column {
            Text(title, style = MaterialTheme.typography.titleSmall)
            Text(text, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

// phones name and place the screen casting page differently
private fun openCastSettings(context: Context) {
    val tries = listOf(Settings.ACTION_CAST_SETTINGS, "android.settings.WIFI_DISPLAY_SETTINGS", Settings.ACTION_SETTINGS)
    for (action in tries) {
        try {
            context.startActivity(Intent(action))
            return
        } catch (_: ActivityNotFoundException) {
        } catch (_: SecurityException) {
        }
    }
}
