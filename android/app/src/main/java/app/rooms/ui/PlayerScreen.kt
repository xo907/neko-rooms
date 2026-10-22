package app.rooms.ui

import android.app.Activity
import android.content.Context
import android.content.ContextWrapper
import android.content.pm.ActivityInfo
import android.content.res.Configuration
import android.widget.Toast
import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.foundation.background
import androidx.compose.foundation.gestures.detectDragGestures
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
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
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.Fullscreen
import androidx.compose.material.icons.outlined.FullscreenExit
import androidx.compose.material.icons.outlined.Keyboard
import androidx.compose.material.icons.outlined.Mouse
import androidx.compose.material.icons.outlined.MoreHoriz
import androidx.compose.material.icons.outlined.PanTool
import androidx.compose.material.icons.outlined.TouchApp
import androidx.compose.material.icons.automirrored.outlined.VolumeOff
import androidx.compose.material.icons.automirrored.outlined.VolumeUp
import androidx.compose.material3.AlertDialog
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
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.changedToDown
import androidx.compose.ui.input.pointer.changedToUp
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.input.pointer.positionChanged
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
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
import org.webrtc.RendererCommon
import org.webrtc.SurfaceViewRenderer
import org.webrtc.VideoTrack
import kotlin.math.abs
import kotlin.math.min

@Composable
fun PlayerScreen(model: AppModel, name: String, invite: String?, nickname: String?, onBack: () -> Unit) {
    val context = LocalContext.current
    val activity = remember(context) { context.findActivity() }
    val view = LocalView.current
    remember { WebRtc.init(context) }
    val client = remember { NekoClient(WatchPartyApp.instance.http, WatchPartyApp.instance.cookies) }
    val st by client.state.collectAsState()

    var room by remember { mutableStateOf<Room?>(null) }
    var joinError by remember { mutableStateOf<String?>(null) }
    var touchPref by remember { mutableStateOf(model.prefs.touchInput) }
    var keyboard by remember { mutableStateOf(false) }
    var showChat by remember { mutableStateOf(true) }
    var controls by remember { mutableStateOf(true) }
    var poke by remember { mutableLongStateOf(0L) }
    var members by remember { mutableStateOf(false) }
    var ctrl by remember { mutableStateOf(false) }
    val landscape = LocalConfiguration.current.orientation == Configuration.ORIENTATION_LANDSCAPE
    val touch = st.touchSupported && touchPref

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
        if (landscape) toggleFullscreen() else onBack()
    }

    var lastHint by remember { mutableLongStateOf(0L) }
    val notHost = {
        val now = System.currentTimeMillis()
        if (now - lastHint > 4000) {
            lastHint = now
            Toast.makeText(context, "Take control to use the browser", Toast.LENGTH_SHORT).show()
        }
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

    Box(Modifier.fillMaxSize().background(Color.Black)) {
        if (landscape) {
            Stage(st, client, touch, notHost, joinError, onBack, Modifier.fillMaxSize())

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
                        IconButton(onClick = { showChat = !showChat; poke++ }) {
                            Icon(Icons.Outlined.ChatBubbleOutline, "Chat", tint = if (showChat) MaterialTheme.colorScheme.primary else Color.White)
                        }
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
            }
            AnimatedVisibility(
                showChat,
                enter = slideInHorizontally { it },
                exit = slideOutHorizontally { it },
                modifier = Modifier.align(Alignment.CenterEnd),
            ) {
                ChatPanel(
                    st, photoOf, onSend = client::sendChat, onMembers = { members = true },
                    modifier = Modifier
                        .fillMaxHeight()
                        .width(320.dp)
                        .background(MaterialTheme.colorScheme.surface.copy(alpha = 0.94f))
                        .imePadding(),
                )
            }
            if (keyboard) {
                KeyboardBar(client, ctrl, { ctrl = it }, onClose = { keyboard = false }, modifier = Modifier.align(Alignment.BottomCenter).imePadding())
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
                        Box(Modifier.padding(end = 12.dp)) {
                            TextButton(onClick = { members = true }) {
                                AvatarStack(st.online.map { it.name }, photoOf, 24.dp, 3)
                            }
                        }
                    }
                }
                val ratio = st.screenWidth.toFloat() / st.screenHeight.coerceAtLeast(1)
                Stage(st, client, touch, notHost, joinError, onBack, Modifier.fillMaxWidth().aspectRatio(ratio))
                bar()
                if (keyboard) KeyboardBar(client, ctrl, { ctrl = it }, onClose = { keyboard = false })
                ChatPanel(
                    st, photoOf, onSend = client::sendChat, onMembers = { members = true },
                    modifier = Modifier.weight(1f).fillMaxWidth().navigationBarsPadding().imePadding(),
                )
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
private fun HostLine(st: RoomState) {
    val text = when {
        st.phase == Phase.Reconnecting -> "Reconnecting…"
        st.isHost -> "You have control"
        st.hostName != null -> "${st.hostName} has control"
        else -> "${st.online.size} watching"
    }
    Text(text, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
}

// Stage shows the video and turns touches into remote input
@Composable
private fun Stage(
    st: RoomState,
    client: NekoClient,
    touch: Boolean,
    onNotHost: () -> Unit,
    joinError: String?,
    onBack: () -> Unit,
    modifier: Modifier,
) {
    var size by remember { mutableStateOf(IntSize.Zero) }
    val sw = st.screenWidth.coerceAtLeast(1)
    val sh = st.screenHeight.coerceAtLeast(1)

    // view position to remote screen position, the video is letterboxed
    fun map(p: Offset): Pair<Int, Int> {
        val scale = min(size.width / sw.toFloat(), size.height / sh.toFloat())
        val ox = (size.width - sw * scale) / 2f
        val oy = (size.height - sh * scale) / 2f
        val x = ((p.x - ox) / scale).toInt().coerceIn(0, sw - 1)
        val y = ((p.y - oy) / scale).toInt().coerceIn(0, sh - 1)
        return x to y
    }

    val input = if (touch) {
        Modifier.pointerInput(st.isHost, size, sw, sh) {
            awaitPointerEventScope {
                while (true) {
                    val event = awaitPointerEvent()
                    if (!st.isHost) {
                        if (event.changes.any { it.changedToDown() }) onNotHost()
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
        // mouse mode: tap clicks, long press right clicks, drag scrolls
        Modifier
            .pointerInput(st.isHost, size, sw, sh) {
                detectTapGestures(
                    onTap = { p -> if (st.isHost) map(p).let { (x, y) -> client.click(x, y) } else onNotHost() },
                    onDoubleTap = { p ->
                        if (st.isHost) map(p).let { (x, y) -> client.click(x, y); client.click(x, y) } else onNotHost()
                    },
                    onLongPress = { p -> if (st.isHost) map(p).let { (x, y) -> client.click(x, y, Buttons.RIGHT) } },
                )
            }
            .pointerInput(st.isHost, size, sw, sh) {
                var acc = 0f
                detectDragGestures(
                    onDragStart = { p -> acc = 0f; if (st.isHost) map(p).let { (x, y) -> client.move(x, y) } },
                    onDrag = { change, drag ->
                        change.consume()
                        if (!st.isHost) return@detectDragGestures
                        acc += drag.y
                        val step = 40f
                        while (abs(acc) >= step) {
                            // finger down scrolls the page up
                            client.scroll(0, if (acc > 0) 1 else -1)
                            acc -= if (acc > 0) step else -step
                        }
                    },
                )
            }
    }

    Box(modifier.background(Color.Black).onSizeChanged { size = it }) {
        VideoView(st.video, Modifier.fillMaxSize())
        Box(Modifier.fillMaxSize().then(input))

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

@Composable
private fun VideoView(track: VideoTrack?, modifier: Modifier) {
    var renderer by remember { mutableStateOf<SurfaceViewRenderer?>(null) }
    AndroidView(
        factory = { ctx ->
            SurfaceViewRenderer(ctx).apply {
                init(WebRtc.egl.eglBaseContext, null)
                setScalingType(RendererCommon.ScalingType.SCALE_ASPECT_FIT)
                setEnableHardwareScaler(true)
                renderer = this
            }
        },
        onRelease = { it.release() },
        modifier = modifier,
    )
    DisposableEffect(track, renderer) {
        val r = renderer
        if (track != null && r != null) runCatching { track.addSink(r) }
        onDispose {
            if (track != null && r != null) runCatching { track.removeSink(r) }
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
    modifier: Modifier,
) {
    val list = rememberLazyListState()
    var text by remember { mutableStateOf("") }
    LaunchedEffect(st.chat.size) {
        if (st.chat.isNotEmpty()) list.animateScrollToItem(st.chat.size - 1)
    }
    Column(modifier) {
        Row(Modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 8.dp), verticalAlignment = Alignment.CenterVertically) {
            Text("Chat", style = MaterialTheme.typography.titleSmall)
            Spacer(Modifier.weight(1f))
            TextButton(onClick = onMembers) { Text("${st.online.size} here") }
        }
        LazyColumn(
            state = list,
            modifier = Modifier.weight(1f).fillMaxWidth(),
            contentPadding = androidx.compose.foundation.layout.PaddingValues(horizontal = 16.dp, vertical = 4.dp),
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
