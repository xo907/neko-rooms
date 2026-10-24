package app.rooms.ui

import android.os.SystemClock
import androidx.annotation.OptIn
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Forward10
import androidx.compose.material.icons.outlined.Pause
import androidx.compose.material.icons.outlined.PlayArrow
import androidx.compose.material.icons.outlined.Replay10
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Slider
import androidx.compose.material3.SliderDefaults
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem
import androidx.media3.common.MimeTypes
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.AspectRatioFrameLayout
import androidx.media3.ui.PlayerView
import app.rooms.neko.LiveClient
import app.rooms.neko.RewindIndex
import app.rooms.neko.streamPositions
import kotlinx.coroutines.delay

// clock is a duration as m:ss
private fun clock(ms: Long): String {
    val s = (ms / 1000).coerceAtLeast(0)
    return "%d:%02d".format(s / 60, s % 60)
}

// RewindPlayer plays the last minutes of the room over the live video, from
// the server's recording, until it catches up with the room. onClose gets a
// message to show, if any.
@OptIn(UnstableApi::class)
@Composable
fun RewindPlayer(live: LiveClient, start: RewindIndex, onClose: (String?) -> Unit, modifier: Modifier = Modifier) {
    val context = LocalContext.current
    val close by rememberUpdatedState(onClose)
    val player = remember { ExoPlayer.Builder(context).build() }
    DisposableEffect(Unit) { onDispose { player.release() } }

    var index by remember { mutableStateOf(start) }
    var indexAt by remember { mutableLongStateOf(SystemClock.elapsedRealtime()) }
    var from by remember { mutableIntStateOf(-1) } // first segment of the stream
    var position by remember { mutableLongStateOf(0L) }
    var playing by remember { mutableStateOf(true) }
    var speed by remember { mutableFloatStateOf(1f) }
    var dragging by remember { mutableStateOf<Float?>(null) }

    fun liveNow() = index.now + (SystemClock.elapsedRealtime() - indexAt)

    // where the stream is on the recording timeline
    fun timeline(pos: Long): Long {
        val segs = index.segments.filter { it.n >= from }
        if (segs.isEmpty()) return liveNow()
        val at = streamPositions(segs)
        var i = 0
        while (i + 1 < segs.size && at[i + 1] <= pos) i++
        return segs[i].start + (pos - at[i])
    }

    // play from about time t of the recording, from the segment it is in
    fun playFrom(t: Long) {
        val seg = index.segments.firstOrNull { it.end > t }
        if (seg == null) {
            close("You are caught up")
            return
        }
        from = seg.n
        position = 0
        player.setMediaItem(MediaItem.Builder().setUri(live.rewindStream(seg.n)).setMimeType(MimeTypes.VIDEO_WEBM).build())
        player.prepare()
        player.playWhenReady = true
    }

    DisposableEffect(player) {
        val listener = object : Player.Listener {
            override fun onPlaybackStateChanged(state: Int) {
                if (state == Player.STATE_ENDED) close("You are caught up")
            }

            override fun onIsPlayingChanged(isPlaying: Boolean) {
                playing = player.playWhenReady
            }

            override fun onPlayerError(error: PlaybackException) {
                close("Rewind stopped: ${error.errorCodeName}")
            }
        }
        player.addListener(listener)
        onDispose { player.removeListener(listener) }
    }

    // a couple of minutes back to start with
    LaunchedEffect(Unit) { playFrom(maxOf(start.first, liveNow() - 120_000)) }

    // newer segments, and the time of the room
    LaunchedEffect(Unit) {
        while (true) {
            delay(5000)
            live.rewindIndex()?.let {
                index = it
                indexAt = SystemClock.elapsedRealtime()
            }
        }
    }

    // follow the position, back to the room once caught up
    LaunchedEffect(Unit) {
        while (true) {
            delay(400)
            position = player.currentPosition
            if (from >= 0 && player.isPlaying && liveNow() - timeline(position) < 7000) {
                close("You are caught up")
                return@LaunchedEffect
            }
        }
    }

    val at = if (from >= 0) timeline(position) else liveNow()
    val now = liveNow()
    val behind = now - (dragging?.toLong() ?: at)

    Box(modifier.background(Color.Black)) {
        AndroidView(
            factory = { ctx ->
                PlayerView(ctx).apply {
                    useController = false
                    resizeMode = AspectRatioFrameLayout.RESIZE_MODE_FIT
                    setShutterBackgroundColor(android.graphics.Color.BLACK)
                    this.player = player
                }
            },
            modifier = Modifier.fillMaxSize(),
        )

        Surface(
            shape = RoundedCornerShape(50),
            color = Color.Black.copy(alpha = 0.7f),
            contentColor = Color.White,
            modifier = Modifier.align(Alignment.TopCenter).padding(top = 10.dp),
        ) {
            Text("⏪ ${clock(behind)} behind", Modifier.padding(horizontal = 12.dp, vertical = 5.dp), style = MaterialTheme.typography.labelLarge)
        }

        Row(
            Modifier
                .align(Alignment.BottomCenter)
                .fillMaxWidth()
                .padding(8.dp)
                .background(Color.Black.copy(alpha = 0.7f), RoundedCornerShape(50))
                .padding(horizontal = 4.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = { player.playWhenReady = !player.playWhenReady; playing = player.playWhenReady }, modifier = Modifier.size(40.dp)) {
                Icon(if (playing) Icons.Outlined.Pause else Icons.Outlined.PlayArrow, if (playing) "Pause" else "Play", tint = Color.White)
            }
            IconButton(onClick = { playFrom(maxOf(index.first, at - 10_000)) }, modifier = Modifier.size(40.dp)) {
                Icon(Icons.Outlined.Replay10, "Back 10 seconds", tint = Color.White)
            }
            IconButton(onClick = { playFrom(at + 10_000) }, modifier = Modifier.size(40.dp)) {
                Icon(Icons.Outlined.Forward10, "Ahead 10 seconds", tint = Color.White)
            }
            TextButton(onClick = {
                speed = when (speed) { 1f -> 1.5f; 1.5f -> 2f; else -> 1f }
                player.setPlaybackSpeed(speed)
            }) { Text(if (speed == 1f) "1×" else "$speed×", color = Color.White) }
            Slider(
                value = (dragging ?: at.toFloat()).coerceIn(index.first.toFloat(), now.toFloat().coerceAtLeast(index.first + 1f)),
                onValueChange = { dragging = it },
                onValueChangeFinished = {
                    dragging?.let { playFrom(it.toLong()) }
                    dragging = null
                },
                valueRange = index.first.toFloat()..now.toFloat().coerceAtLeast(index.first + 1f),
                colors = SliderDefaults.colors(inactiveTrackColor = Color.White.copy(alpha = 0.25f)),
                modifier = Modifier.weight(1f).padding(horizontal = 6.dp),
            )
            Button(
                onClick = { close(null) },
                contentPadding = ButtonDefaults.TextButtonContentPadding,
                modifier = Modifier.padding(end = 4.dp),
            ) { Text("Live") }
        }
    }
}

private data class CaptionLine(val key: Long, val text: String, val at: Long)

// Captions shows the last subtitle lines over the video, each for a few
// seconds
@Composable
fun Captions(live: LiveClient, modifier: Modifier = Modifier) {
    val lines = remember { mutableStateListOf<CaptionLine>() }
    var key by remember { mutableLongStateOf(0L) }
    LaunchedEffect(live) {
        live.captionLines.collect { text ->
            lines.add(CaptionLine(key++, text, SystemClock.elapsedRealtime()))
            while (lines.size > 2) lines.removeAt(0)
        }
    }
    LaunchedEffect(live) {
        while (true) {
            delay(500)
            val now = SystemClock.elapsedRealtime()
            lines.removeAll { now - it.at > 7000 }
        }
    }
    if (lines.isEmpty()) return
    Column(
        modifier.padding(bottom = 14.dp, start = 16.dp, end = 16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(3.dp),
    ) {
        lines.forEach { l ->
            Text(
                l.text,
                color = Color.White,
                fontSize = 15.sp,
                fontWeight = FontWeight.Medium,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .widthIn(max = 560.dp)
                    .background(Color.Black.copy(alpha = 0.75f), RoundedCornerShape(6.dp))
                    .padding(horizontal = 8.dp, vertical = 2.dp),
            )
        }
    }
}

// CaptionsMenu picks the subtitle language
@Composable
fun CaptionsMenu(live: LiveClient, open: Boolean, onDismiss: () -> Unit, onPick: (String) -> Unit) {
    val ls by live.state.collectAsState()
    DropdownMenu(expanded = open, onDismissRequest = onDismiss) {
        (listOf("" to "Off") + ls.captionLangs.map { it.code to it.name }).forEach { (code, name) ->
            DropdownMenuItem(
                text = {
                    Text(
                        name,
                        color = if (code == ls.captions) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
                    )
                },
                onClick = {
                    onDismiss()
                    if (code != ls.captions) onPick(code)
                },
            )
        }
    }
}
