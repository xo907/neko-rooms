package app.rooms.ui

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.graphics.SurfaceTexture
import android.view.TextureView
import android.widget.Toast
import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.Animatable
import androidx.compose.animation.core.tween
import androidx.compose.animation.expandHorizontally
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.shrinkHorizontally
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.awaitEachGesture
import androidx.compose.foundation.gestures.awaitFirstDown
import androidx.compose.foundation.gestures.calculateCentroid
import androidx.compose.foundation.gestures.calculatePan
import androidx.compose.foundation.gestures.calculateZoom
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.Send
import androidx.compose.material.icons.automirrored.outlined.VolumeOff
import androidx.compose.material.icons.automirrored.outlined.VolumeUp
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.Fullscreen
import androidx.compose.material.icons.outlined.FullscreenExit
import androidx.compose.material.icons.outlined.Keyboard
import androidx.compose.material.icons.outlined.Mouse
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.PanTool
import androidx.compose.material.icons.outlined.TouchApp
import androidx.compose.material.icons.outlined.ZoomOutMap
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.FilledTonalButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.IconButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.input.pointer.PointerEvent
import androidx.compose.ui.input.pointer.changedToDown
import androidx.compose.ui.input.pointer.changedToUp
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChanged
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalHapticFeedback
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.TextRange
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.TextFieldValue
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.core.view.WindowCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.WindowInsetsControllerCompat
import app.rooms.WatchPartyApp
import app.rooms.data.Room
import app.rooms.neko.Buttons
import app.rooms.neko.ChatLine
import app.rooms.neko.Keys
import app.rooms.neko.NekoClient
import app.rooms.neko.Phase
import app.rooms.neko.RoomState
import app.rooms.neko.WebRtc
import kotlinx.coroutines.delay
import org.webrtc.EglBase
import org.webrtc.EglRenderer
import org.webrtc.GlRectDrawer
import org.webrtc.VideoTrack
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import kotlin.math.abs
import kotlin.math.max
import kotlin.math.min

@Composable
fun PlayerScreen(model: AppModel, name: String, invite: String?, nickname: String?, onBack: () -> Unit) {
    val context = LocalContext.current
    val activity = remember(context) { context.findActivity() }
    val view = LocalView.current
    val focusManager = LocalFocusManager.current
    remember { WebRtc.init(context) }
    val client = remember { NekoClient(WatchPartyApp.instance.http, WatchPartyApp.instance.cookies) }
    val st by client.state.collectAsState()

    var room by remember { mutableStateOf<Room?>(null) }
    var joinError by remember { mutableStateOf<String?>(null) }
    var touchPref by remember { mutableStateOf(model.prefs.touchInput) }
    var keyboard by remember { mutableStateOf(false) }
    var controls by remember { mutableStateOf(true) }
    var poke by remember { mutableLongStateOf(0L) }
    var members by remember { mutableStateOf(false) }
    var ctrl by remember { mutableStateOf(false) }
    val landscape = LocalConfiguration.current.orientation == Configuration.ORIENTATION_LANDSCAPE
    val touch = st.touchSupported && touchPref

    // chat is open below the video in portrait, closed beside it in landscape
    var portraitChat by remember { mutableStateOf(true) }
    var sideChat by remember { mutableStateOf(false) }
    val chatOpen = if (landscape) sideChat else portraitChat
    fun setChat(open: Boolean) {
        if (landscape) sideChat = open else portraitChat = open
        if (!open) focusManager.clearFocus()
        poke++
    }
    var seen by remember { mutableIntStateOf(0) }
    LaunchedEffect(chatOpen, st.chat.size) { if (chatOpen) seen = st.chat.size }
    val unread = if (chatOpen) 0 else st.chat.drop(seen).count { !it.system && !it.mine }

    LaunchedEffect(Unit) {
        runCatching { model.api.room(name, invite) }.onSuccess { room = it }
        runCatching { model.api.join(name, invite, nickname) }
            .onSuccess { client.start(it) }
            .onFailure { joinError = it.message ?: "Could not join the room" }
    }

    DisposableEffect(Unit) {
        view.keepScreenOn = true
        onDispose {
            client.stop()
            view.keepScreenOn = false
            activity?.requestedOrientation = ActivityInfo.SCREEN_ORIENTATION_UNSPECIFIED
        }
    }

    // immersive while the video fills a landscape screen
    DisposableEffect(landscape) {
        val window = activity?.window
        if (window != null) {
            val insets = WindowCompat.getInsetsController(window, view)
            if (landscape) {
                insets.systemBarsBehavior = WindowInsetsControllerCompat.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE
                insets.hide(WindowInsetsCompat.Type.systemBars())
            } else {
                insets.show(WindowInsetsCompat.Type.systemBars())
            }
        }
        onDispose {
            if (window != null) WindowCompat.getInsetsController(window, view).show(WindowInsetsCompat.Type.systemBars())
        }
    }

    // landscape controls hide after a few quiet seconds
    LaunchedEffect(controls, poke, landscape) {
        if (landscape && controls) {
            delay(4000)
            controls = false
        }
    }

    fun toggleFullscreen() {
        activity?.requestedOrientation =
            if (landscape) ActivityInfo.SCREEN_ORIENTATION_PORTRAIT else ActivityInfo.SCREEN_ORIENTATION_SENSOR_LANDSCAPE
    }

    BackHandler {
        when {
            landscape && sideChat -> setChat(false)
            landscape -> toggleFullscreen()
            else -> onBack()
        }
    }

    // touching the browser without control takes it when nobody has it
    var lastHint by remember { mutableLongStateOf(0L) }
    val notHost = {
        if (st.phase == Phase.Live && st.hostId == null) {
            client.requestControl()
        } else {
            val now = System.currentTimeMillis()
            if (now - lastHint > 4000) {
                lastHint = now
                val who = st.hostName ?: "Someone"
                Toast.makeText(context, "$who has control. Tap Take control to ask for it", Toast.LENGTH_SHORT).show()
            }
        }
    }
    // a touch on the video puts away the chat keyboard
    val onStageTouch = {
        if (!keyboard) focusManager.clearFocus()
        poke++
    }

    val title = room?.displayTitle ?: name
    val photoOf = { n: String -> room?.avatars?.get(n)?.let { model.api.apiUrl(it) } }
    val bar = @Composable {
        ControlBar(
            st = st,
            touch = touch,
            keyboard = keyboard,
            landscape = landscape,
            onControl = { if (st.isHost) client.releaseControl() else client.requestControl() },
            onKeyboard = { keyboard = !keyboard },
            onMute = { client.setMuted(!st.muted) },
            onInputMode = if (st.touchSupported) ({
                touchPref = !touchPref
                model.prefs.touchInput = touchPref
            }) else null,
            onFullscreen = { toggleFullscreen() },
        )
    }
    val chat = @Composable { modifier: Modifier ->
        ChatPanel(st, photoOf, onSend = client::sendChat, onMembers = { members = true }, onClose = { setChat(false) }, modifier = modifier)
    }

    Box(Modifier.fillMaxSize().background(Color.Black)) {
        if (landscape) {
            Row(Modifier.fillMaxSize()) {
                Box(Modifier.weight(1f).fillMaxHeight()) {
                    Stage(st, client, touch, notHost, onStageTouch, joinError, onBack, Modifier.fillMaxSize())

                    AnimatedVisibility(controls, enter = fadeIn(), exit = fadeOut(), modifier = Modifier.align(Alignment.TopCenter)) {
                        Column(
                            Modifier
                                .fillMaxWidth()
                                .background(Brush.verticalGradient(listOf(Color.Black.copy(alpha = 0.75f), Color.Transparent)))
                                .padding(horizontal = 8.dp, vertical = 6.dp)
                        ) {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                IconButton(onClick = { toggleFullscreen() }) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Exit fullscreen", tint = Color.White) }
                                Column(Modifier.weight(1f)) {
                                    Text(title, color = Color.White, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                                    HostLine(st)
                                }
                                ChatButton(sideChat, unread, Color.White) { setChat(!sideChat) }
                            }
                            bar()
                        }
                    }
                    if (!controls) {
                        IconButton(
                            onClick = { controls = true; poke++ },
                            colors = IconButtonDefaults.iconButtonColors(containerColor = Color.Black.copy(alpha = 0.45f)),
                            modifier = Modifier.align(Alignment.TopStart).padding(10.dp).size(36.dp),
                        ) { Icon(Icons.Outlined.MoreHoriz, "Show controls", tint = Color.White) }
                        if (unread > 0) {
                            Box(Modifier.align(Alignment.TopEnd).padding(6.dp)) {
                                ChatButton(false, unread, Color.White) { setChat(true) }
                            }
                        }
                    }
                    if (keyboard) {
                        KeyboardBar(client, ctrl, { ctrl = it }, onClose = { keyboard = false }, modifier = Modifier.align(Alignment.BottomCenter).imePadding())
                    }
                }
                // the chat sits beside the video instead of over it
                AnimatedVisibility(sideChat, enter = expandHorizontally(), exit = shrinkHorizontally()) {
                    chat(
                        Modifier
                            .fillMaxHeight()
                            .width(300.dp)
                            .background(MaterialTheme.colorScheme.surface)
                            .imePadding()
                    )
                }
            }
        } else {
            Column(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                Row(
                    Modifier.statusBarsPadding().padding(horizontal = 4.dp, vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Leave") }
                    Column(Modifier.weight(1f)) {
                        Text(title, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                        HostLine(st)
                    }
                    if (st.online.isNotEmpty()) {
                        TextButton(onClick = { members = true }) {
                            AvatarStack(st.online.map { it.name }, photoOf, 24.dp, 3)
                        }
                    }
                    ChatButton(portraitChat, unread, MaterialTheme.colorScheme.onSurface) { setChat(!portraitChat) }
                }
                val ratio = st.screenWidth.toFloat() / st.screenHeight.coerceAtLeast(1)
                // without the chat the video gets all the room there is
                val stage = if (portraitChat) Modifier.fillMaxWidth().aspectRatio(ratio) else Modifier.fillMaxWidth().weight(1f)
                Stage(st, client, touch, notHost, onStageTouch, joinError, onBack, stage)
                bar()
                if (keyboard) KeyboardBar(client, ctrl, { ctrl = it }, onClose = { keyboard = false })
                if (portraitChat) {
                    chat(Modifier.weight(1f).fillMaxWidth().navigationBarsPadding().imePadding())
                } else {
                    Spacer(Modifier.navigationBarsPadding().imePadding())
                }
            }
        }

        // hidden field that receives the on-screen keyboard
        if (keyboard) RemoteKeyboard(client, ctrl) { ctrl = false }
    }

    if (members) {
        AlertDialog(
            onDismissRequest = { members = false },
            title = { Text("In this room · ${st.online.size}") },
            text = {
                Column {
                    st.online.forEach { m ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 6.dp)) {
                            Avatar(m.name, photoOf(m.name), 30.dp)
                            Spacer(Modifier.width(12.dp))
                            Text(m.name + if (m.id == st.myId) " (you)" else "", modifier = Modifier.weight(1f))
                            if (m.id == st.hostId) Tag("Control", highlight = true)
                        }
                    }
                }
            },
            confirmButton = { TextButton(onClick = { members = false }) { Text("Close") } },
        )
    }
}

@Composable
private fun ChatButton(open: Boolean, unread: Int, tint: Color, onClick: () -> Unit) {
    IconButton(onClick = onClick) {
        BadgedBox(badge = { if (unread > 0) Badge { Text(if (unread > 9) "9+" else "$unread") } }) {
            Icon(
                Icons.Outlined.ChatBubbleOutline,
                if (open) "Hide chat" else "Show chat",
                tint = if (open) MaterialTheme.colorScheme.primary else tint,
            )
        }
    }
}

@Composable
private fun HostLine(st: RoomState) {
    val text = when {
        st.phase == Phase.Reconnecting -> "Reconnecting…"
        st.isHost -> "You have control"
        st.hostName != null -> "${st.hostName} has control"
        else -> "${st.online.size} watching"
    }
    Text(text, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
}

private enum class Gesture { Pending, Held, Drag, Scroll, Zoom }

// Stage shows the video and turns touches into remote input.
//
// Mouse mode, made for fingers: tap clicks (tap twice to double click), drag
// scrolls, hold then drag selects or drags, hold and let go right clicks,
// two fingers zoom and pan the picture.
@Composable
private fun Stage(
    st: RoomState,
    client: NekoClient,
    touch: Boolean,
    onNotHost: () -> Unit,
    onTouchStart: () -> Unit,
    joinError: String?,
    onBack: () -> Unit,
    modifier: Modifier,
) {
    val density = LocalDensity.current
    val haptics = LocalHapticFeedback.current
    var size by remember { mutableStateOf(IntSize.Zero) }
    val sw = st.screenWidth.coerceAtLeast(1)
    val sh = st.screenHeight.coerceAtLeast(1)
    var zoom by remember { mutableFloatStateOf(1f) }
    var pan by remember { mutableStateOf(Offset.Zero) }
    var pulseAt by remember { mutableStateOf<Offset?>(null) }
    var pulseKey by remember { mutableIntStateOf(0) }

    val isHost by rememberUpdatedState(st.isHost)
    val notHost by rememberUpdatedState(onNotHost)
    val touchStart by rememberUpdatedState(onTouchStart)

    LaunchedEffect(sw, sh) {
        zoom = 1f
        pan = Offset.Zero
    }

    // size of the letterboxed video at zoom 1
    val fit = if (size.width > 0 && size.height > 0) min(size.width / sw.toFloat(), size.height / sh.toFloat()) else 0f
    val vw = sw * fit
    val vh = sh * fit

    fun clampPan(p: Offset, z: Float): Offset {
        val mx = max(0f, vw * z / 2f - size.width / 2f)
        val my = max(0f, vh * z / 2f - size.height / 2f)
        return Offset(p.x.coerceIn(-mx, mx), p.y.coerceIn(-my, my))
    }

    // view position to remote screen position, through letterbox and zoom
    fun map(p: Offset): Pair<Int, Int> {
        if (fit <= 0f) return 0 to 0
        val u = (p.x - size.width / 2f - pan.x) / zoom + vw / 2f
        val v = (p.y - size.height / 2f - pan.y) / zoom + vh / 2f
        val x = (u / fit).toInt().coerceIn(0, sw - 1)
        val y = (v / fit).toInt().coerceIn(0, sh - 1)
        return x to y
    }

    fun zoomBy(event: PointerEvent) {
        val c = event.calculateCentroid()
        if (c == Offset.Unspecified) return
        val z2 = (zoom * event.calculateZoom()).coerceIn(1f, 4f)
        val center = Offset(size.width / 2f, size.height / 2f)
        // keep the point between the fingers under the fingers
        val next = (c - center) - (c - center - pan) * (z2 / zoom) + event.calculatePan()
        zoom = z2
        pan = clampPan(next, z2)
    }

    val input = if (touch) {
        Modifier.pointerInput(sw, sh, size) {
            awaitPointerEventScope {
                while (true) {
                    val event = awaitPointerEvent()
                    if (event.changes.any { it.changedToDown() }) touchStart()
                    if (!isHost) {
                        if (event.changes.any { it.changedToDown() }) notHost()
                        continue
                    }
                    event.changes.forEach { c ->
                        val id = (c.id.value and 0xffff).toInt()
                        val (x, y) = map(c.position)
                        when {
                            c.changedToDown() -> client.touchBegin(id, x, y)
                            c.changedToUp() -> client.touchEnd(id, x, y)
                            c.pressed && c.positionChanged() -> client.touchUpdate(id, x, y)
                        }
                        c.consume()
                    }
                }
            }
        }
    } else {
        Modifier.pointerInput(sw, sh, size) {
            val step = 22.dp.toPx()
            awaitEachGesture {
                val down = awaitFirstDown(requireUnconsumed = false)
                touchStart()
                val start = down.position
                val slop = viewConfiguration.touchSlop
                val holdAt = down.uptimeMillis + viewConfiguration.longPressTimeoutMillis
                var now = down.uptimeMillis
                var mode = Gesture.Pending
                var last = start
                var acc = Offset.Zero

                while (true) {
                    val event = if (mode == Gesture.Pending) {
                        withTimeoutOrNull((holdAt - now).coerceAtLeast(1)) { awaitPointerEvent() }
                    } else {
                        awaitPointerEvent()
                    }
                    if (event == null) {
                        mode = Gesture.Held
                        if (isHost) haptics.performHapticFeedback(HapticFeedbackType.LongPress)
                        continue
                    }
                    now = event.changes.first().uptimeMillis
                    val pressed = event.changes.filter { it.pressed }

                    if (pressed.size >= 2) {
                        if (mode == Gesture.Drag) client.buttonUp(Buttons.LEFT)
                        mode = Gesture.Zoom
                        zoomBy(event)
                        event.changes.forEach { it.consume() }
                        continue
                    }
                    if (pressed.isEmpty()) {
                        when (mode) {
                            Gesture.Pending -> if (isHost) {
                                val (x, y) = map(start)
                                client.click(x, y)
                                pulseAt = start
                                pulseKey++
                            } else {
                                notHost()
                            }
                            Gesture.Held -> if (isHost) {
                                val (x, y) = map(start)
                                client.click(x, y, Buttons.RIGHT)
                            } else {
                                notHost()
                            }
                            Gesture.Drag -> client.buttonUp(Buttons.LEFT)
                            else -> {}
                        }
                        event.changes.forEach { it.consume() }
                        break
                    }

                    val c = pressed.first()
                    val pos = c.position
                    when (mode) {
                        Gesture.Pending -> if ((pos - start).getDistance() > slop) {
                            mode = Gesture.Scroll
                            last = pos
                        }
                        Gesture.Scroll -> {
                            if (isHost) {
                                // the page follows the finger, like scrolling on a phone
                                acc += pos - last
                                while (abs(acc.y) >= step) {
                                    val dir = if (acc.y > 0) 1 else -1
                                    client.scroll(0, dir)
                                    acc = acc.copy(y = acc.y - dir * step)
                                }
                                while (abs(acc.x) >= step) {
                                    val dir = if (acc.x > 0) 1 else -1
                                    client.scroll(dir, 0)
                                    acc = acc.copy(x = acc.x - dir * step)
                                }
                            } else if (zoom > 1f) {
                                pan = clampPan(pan + (pos - last), zoom)
                            }
                            last = pos
                        }
                        Gesture.Held -> if (isHost && (pos - start).getDistance() > slop / 2) {
                            mode = Gesture.Drag
                            val (sx, sy) = map(start)
                            client.move(sx, sy)
                            client.buttonDown(Buttons.LEFT)
                            val (x, y) = map(pos)
                            client.move(x, y)
                        }
                        Gesture.Drag -> {
                            val (x, y) = map(pos)
                            client.move(x, y)
                        }
                        Gesture.Zoom -> {}
                    }
                    c.consume()
                }
            }
        }
    }

    Box(modifier.background(Color.Black).clipToBounds().onSizeChanged { size = it }) {
        VideoView(
            st.video,
            Modifier
                .align(Alignment.Center)
                .size(with(density) { vw.toDp() }, with(density) { vh.toDp() })
                .graphicsLayer {
                    scaleX = zoom
                    scaleY = zoom
                    translationX = pan.x
                    translationY = pan.y
                },
        )
        Box(Modifier.fillMaxSize().then(input))

        // a ring where a tap landed, so a click is felt
        pulseAt?.let { at ->
            val anim = remember(pulseKey) { Animatable(0f) }
            LaunchedEffect(pulseKey) { anim.animateTo(1f, tween(380)) }
            Canvas(Modifier.fillMaxSize()) {
                val t = anim.value
                drawCircle(
                    color = Color.White.copy(alpha = 0.6f * (1f - t)),
                    radius = 8.dp.toPx() + 16.dp.toPx() * t,
                    center = at,
                    style = Stroke(2.dp.toPx()),
                )
            }
        }

        if (zoom > 1.01f) {
            Surface(
                onClick = { zoom = 1f; pan = Offset.Zero },
                shape = RoundedCornerShape(50),
                color = Color.Black.copy(alpha = 0.55f),
                contentColor = Color.White,
                modifier = Modifier.align(Alignment.BottomStart).padding(10.dp),
            ) {
                Row(Modifier.padding(horizontal = 12.dp, vertical = 6.dp), verticalAlignment = Alignment.CenterVertically) {
                    Icon(Icons.Outlined.ZoomOutMap, null, Modifier.size(16.dp))
                    Spacer(Modifier.width(6.dp))
                    Text("Fit", style = MaterialTheme.typography.labelLarge)
                }
            }
        }

        when {
            joinError != null || st.phase == Phase.Ended -> Overlay {
                Text(joinError ?: st.error ?: "The room ended", color = Color.White, textAlign = TextAlign.Center)
                Spacer(Modifier.height(14.dp))
                Button(onClick = onBack) { Text("Back") }
            }
            st.video == null -> Overlay {
                CircularProgressIndicator(color = Color.White, strokeWidth = 2.dp, modifier = Modifier.size(28.dp))
                Spacer(Modifier.height(12.dp))
                Text(if (st.phase == Phase.Reconnecting) "Reconnecting…" else "Joining room…", color = Color.White.copy(alpha = 0.8f))
            }
        }
    }
}

@Composable
private fun Overlay(content: @Composable () -> Unit) {
    Column(
        Modifier.fillMaxSize().background(Color.Black.copy(alpha = 0.7f)).padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) { content() }
}

// VideoView draws the stream on a TextureView, which, unlike a SurfaceView,
// can be zoomed and clipped like any other view
@Composable
private fun VideoView(track: VideoTrack?, modifier: Modifier) {
    val renderer = remember {
        EglRenderer("video").apply { init(WebRtc.egl.eglBaseContext, EglBase.CONFIG_PLAIN, GlRectDrawer()) }
    }
    DisposableEffect(Unit) { onDispose { renderer.release() } }
    AndroidView(
        factory = { ctx ->
            TextureView(ctx).apply {
                surfaceTextureListener = object : TextureView.SurfaceTextureListener {
                    override fun onSurfaceTextureAvailable(surface: SurfaceTexture, width: Int, height: Int) {
                        renderer.createEglSurface(surface)
                    }

                    override fun onSurfaceTextureSizeChanged(surface: SurfaceTexture, width: Int, height: Int) {}

                    override fun onSurfaceTextureDestroyed(surface: SurfaceTexture): Boolean {
                        val done = CountDownLatch(1)
                        renderer.releaseEglSurface { done.countDown() }
                        done.await(1, TimeUnit.SECONDS)
                        return true
                    }

                    override fun onSurfaceTextureUpdated(surface: SurfaceTexture) {}
                }
            }
        },
        modifier = modifier,
    )
    DisposableEffect(track) {
        if (track != null) runCatching { track.addSink(renderer) }
        onDispose {
            if (track != null) runCatching { track.removeSink(renderer) }
        }
    }
}

@Composable
private fun ControlBar(
    st: RoomState,
    touch: Boolean,
    keyboard: Boolean,
    landscape: Boolean,
    onControl: () -> Unit,
    onKeyboard: () -> Unit,
    onMute: () -> Unit,
    onInputMode: (() -> Unit)?,
    onFullscreen: () -> Unit,
) {
    val tint = if (landscape) Color.White else MaterialTheme.colorScheme.onSurface
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 10.dp, vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(4.dp),
    ) {
        FilledTonalButton(onClick = onControl, enabled = st.phase == Phase.Live, shape = RoundedCornerShape(50)) {
            Icon(Icons.Outlined.PanTool, null, Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Text(if (st.isHost) "Release" else "Take control")
        }
        Spacer(Modifier.weight(1f))
        if (st.isHost) {
            IconButton(onClick = onKeyboard) {
                Icon(Icons.Outlined.Keyboard, "Keyboard", tint = if (keyboard) MaterialTheme.colorScheme.primary else tint)
            }
            if (onInputMode != null) {
                IconButton(onClick = onInputMode) {
                    Icon(if (touch) Icons.Outlined.TouchApp else Icons.Outlined.Mouse, if (touch) "Touch input" else "Mouse input", tint = tint)
                }
            }
        }
        IconButton(onClick = onMute) {
            Icon(if (st.muted) Icons.AutoMirrored.Outlined.VolumeOff else Icons.AutoMirrored.Outlined.VolumeUp, "Sound", tint = tint)
        }
        IconButton(onClick = onFullscreen) {
            Icon(if (landscape) Icons.Outlined.FullscreenExit else Icons.Outlined.Fullscreen, "Fullscreen", tint = tint)
        }
    }
}

@Composable
private fun ChatPanel(
    st: RoomState,
    photoOf: (String) -> String?,
    onSend: (String) -> Unit,
    onMembers: () -> Unit,
    onClose: () -> Unit,
    modifier: Modifier,
) {
    val list = rememberLazyListState()
    var text by remember { mutableStateOf("") }
    LaunchedEffect(st.chat.size) {
        if (st.chat.isNotEmpty()) list.animateScrollToItem(st.chat.size - 1)
    }
    Column(modifier) {
        Row(Modifier.fillMaxWidth().padding(start = 16.dp, end = 4.dp, top = 4.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("Chat", style = MaterialTheme.typography.titleSmall)
            Spacer(Modifier.weight(1f))
            TextButton(onClick = onMembers) { Text("${st.online.size} here") }
            IconButton(onClick = onClose) { Icon(Icons.Outlined.Close, "Hide chat") }
        }
        LazyColumn(
            state = list,
            modifier = Modifier.weight(1f).fillMaxWidth(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 4.dp),
            verticalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            if (st.chat.isEmpty()) {
                item {
                    Text("Say hi to the room.", color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodySmall)
                }
            }
            items(st.chat, key = { it.key }) { line -> ChatRow(line, photoOf) }
        }
        Row(Modifier.fillMaxWidth().padding(8.dp), verticalAlignment = Alignment.CenterVertically) {
            TextField(
                value = text,
                onValueChange = { if (it.length <= 500) text = it },
                placeholder = { Text("Send a message") },
                singleLine = true,
                shape = RoundedCornerShape(50),
                colors = TextFieldDefaults.colors(
                    focusedIndicatorColor = Color.Transparent,
                    unfocusedIndicatorColor = Color.Transparent,
                    focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                    unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainer,
                ),
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                keyboardActions = KeyboardActions(onSend = { onSend(text); text = "" }),
                modifier = Modifier.weight(1f),
            )
            IconButton(onClick = { onSend(text); text = "" }, enabled = text.isNotBlank() && st.phase == Phase.Live) {
                Icon(Icons.AutoMirrored.Outlined.Send, "Send", tint = MaterialTheme.colorScheme.primary)
            }
        }
    }
}

@Composable
private fun ChatRow(line: ChatLine, photoOf: (String) -> String?) {
    if (line.system) {
        Text(
            line.text,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.fillMaxWidth(),
        )
        return
    }
    Row {
        Avatar(line.sender, photoOf(line.sender), 28.dp)
        Spacer(Modifier.width(10.dp))
        Column {
            Text(if (line.mine) "You" else line.sender, style = MaterialTheme.typography.labelLarge)
            Text(line.text, style = MaterialTheme.typography.bodyMedium)
        }
    }
}

// KeyboardBar has the keys a phone keyboard lacks
@Composable
private fun KeyboardBar(client: NekoClient, ctrl: Boolean, onCtrl: (Boolean) -> Unit, onClose: () -> Unit, modifier: Modifier = Modifier) {
    Surface(color = MaterialTheme.colorScheme.surfaceContainer, modifier = modifier.fillMaxWidth()) {
        Row(
            Modifier.horizontalScroll(rememberScrollState()).padding(horizontal = 8.dp, vertical = 6.dp),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            fun press(sym: Int) {
                client.key(sym, ctrl)
                onCtrl(false)
            }
            KeyChip("Esc") { press(Keys.ESCAPE) }
            KeyChip("Tab") { press(Keys.TAB) }
            KeyChip("Ctrl", active = ctrl) { onCtrl(!ctrl) }
            KeyChip("←") { press(Keys.LEFT) }
            KeyChip("↑") { press(Keys.UP) }
            KeyChip("↓") { press(Keys.DOWN) }
            KeyChip("→") { press(Keys.RIGHT) }
            KeyChip("⌫") { press(Keys.BACKSPACE) }
            KeyChip("Del") { press(Keys.DELETE) }
            KeyChip("Enter") { press(Keys.RETURN) }
            KeyChip("Done") { onClose() }
        }
    }
}

@Composable
private fun KeyChip(label: String, active: Boolean = false, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(10.dp),
        color = if (active) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceContainerHighest,
        contentColor = if (active) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface,
    ) {
        Text(label, style = MaterialTheme.typography.labelLarge, modifier = Modifier.padding(horizontal = 14.dp, vertical = 9.dp))
    }
}

private const val SENTINEL = "​​"

// RemoteKeyboard keeps an invisible text field focused and forwards what is
// typed. Two zero width characters stay in it, so backspace can be detected.
@Composable
private fun RemoteKeyboard(client: NekoClient, ctrl: Boolean, onCtrlUsed: () -> Unit) {
    val focus = remember { FocusRequester() }
    val keyboard = LocalSoftwareKeyboardController.current
    var value by remember { mutableStateOf(TextFieldValue(SENTINEL, TextRange(SENTINEL.length))) }

    LaunchedEffect(Unit) {
        focus.requestFocus()
        keyboard?.show()
    }
    DisposableEffect(Unit) { onDispose { keyboard?.hide() } }

    BasicTextField(
        value = value,
        onValueChange = { v ->
            val text = v.text
            when {
                text.length < SENTINEL.length -> repeat(SENTINEL.length - text.length) { client.key(Keys.BACKSPACE) }
                else -> text.replace("​", "").forEach { ch ->
                    client.key(Keys.forChar(ch.code), ctrl)
                    if (ctrl) onCtrlUsed()
                }
            }
            value = TextFieldValue(SENTINEL, TextRange(SENTINEL.length))
        },
        keyboardOptions = KeyboardOptions(
            keyboardType = KeyboardType.Password,
            imeAction = ImeAction.Default,
            autoCorrectEnabled = false,
        ),
        modifier = Modifier.size(1.dp).focusRequester(focus),
    )
}

private fun Context.findActivity(): Activity? {
    var c: Context? = this
    while (c is ContextWrapper) {
        if (c is Activity) return c
        c = c.baseContext
    }
    return null
}
