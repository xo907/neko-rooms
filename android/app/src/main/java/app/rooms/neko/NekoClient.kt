package app.rooms.neko

import app.rooms.data.json
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.booleanOrNull
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.webrtc.AudioTrack
import org.webrtc.DataChannel
import org.webrtc.IceCandidate
import org.webrtc.MediaConstraints
import org.webrtc.MediaStream
import org.webrtc.PeerConnection
import org.webrtc.RtpTransceiver
import org.webrtc.SdpObserver
import org.webrtc.SessionDescription
import org.webrtc.VideoTrack
import java.nio.ByteBuffer
import java.util.concurrent.atomic.AtomicLong

data class Member(val id: String, val name: String, val isAdmin: Boolean, val connected: Boolean)

data class ChatLine(val key: Long, val sender: String, val text: String, val system: Boolean, val mine: Boolean)

enum class Phase { Connecting, Live, Reconnecting, Ended }

data class RoomState(
    val phase: Phase = Phase.Connecting,
    val error: String? = null,
    val myId: String? = null,
    val members: Map<String, Member> = emptyMap(),
    val hostId: String? = null,
    val screenWidth: Int = 1280,
    val screenHeight: Int = 720,
    val video: VideoTrack? = null,
    val chat: List<ChatLine> = emptyList(),
    val touchSupported: Boolean = false,
    val muted: Boolean = false,
) {
    val isHost: Boolean get() = myId != null && myId == hostId
    val hostName: String? get() = hostId?.let { members[it]?.name }
    val online: List<Member> get() = members.values.filter { it.connected }.sortedBy { it.name.lowercase() }
}

// NekoClient joins a neko room with the v3 API: login, websocket signaling,
// WebRTC media and a data channel for mouse, keyboard and touch input.
class NekoClient(http: OkHttpClient) {
    // own scope: it outlives the screen long enough to log out cleanly
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    // no cookie jar: neko sets its session cookie on the whole host, a cookie
    // from one room would break logging in to the next. Tokens are explicit.
    private val http = http.newBuilder().cookieJar(CookieJar.NO_COOKIES).build()

    private val _state = MutableStateFlow(RoomState())
    val state: StateFlow<RoomState> = _state

    private var roomBase: HttpUrl? = null
    private var username = ""
    private var password = ""
    private var token: String? = null

    private var ws: WebSocket? = null
    private var pc: PeerConnection? = null
    private var dc: DataChannel? = null
    private var audio: AudioTrack? = null
    private val pending = mutableListOf<IceCandidate>()
    private var remoteSet = false

    private var closed = false
    private var retries = 0
    private var job: Job? = null
    private val keys = AtomicLong()

    fun start(joinUrl: HttpUrl) {
        roomBase = joinUrl.newBuilder().query(null).build()
        username = joinUrl.queryParameter("usr") ?: "guest"
        password = joinUrl.queryParameter("pwd") ?: ""
        connect()
    }

    fun stop() {
        closed = true
        job?.cancel()
        ws?.close(1000, "bye")
        ws = null
        closePeer()
        val t = token
        val base = roomBase
        scope.launch {
            if (t != null && base != null) {
                runCatching {
                    http.newCall(
                        Request.Builder().url(base.resolve("api/logout")!!)
                            .header("Authorization", "Bearer $t")
                            .post("{}".toRequestBody(JSON)).build()
                    ).execute().close()
                }
            }
            scope.cancel()
        }
    }

    private fun connect() {
        job = scope.launch(Dispatchers.IO) {
            try {
                token = login()
                openSocket()
            } catch (e: Exception) {
                fail(e.message ?: "Could not join the room")
            }
        }
    }

    private fun login(): String {
        val base = roomBase!!
        val body = buildJsonObject {
            put("username", username)
            put("password", password)
        }.toString().toRequestBody(JSON)
        http.newCall(Request.Builder().url(base.resolve("api/login")!!).post(body).build()).execute().use { res ->
            val text = res.body?.string().orEmpty()
            if (!res.isSuccessful) {
                throw IllegalStateException(
                    when (res.code) {
                        401 -> "The room did not accept the invite, try again from the room page"
                        403 -> "The room is locked"
                        else -> "Could not join the room (${res.code})"
                    }
                )
            }
            // the token comes as a cookie, or in the body when cookies are off
            val fromCookie = res.headers("Set-Cookie").firstNotNullOfOrNull { c ->
                val pair = c.substringBefore(';')
                if (pair.substringBefore('=').contains("session", ignoreCase = true)) pair.substringAfter('=') else null
            }
            val fromBody = runCatching {
                json.parseToJsonElement(text).jsonObject["token"]?.jsonPrimitive?.contentOrNull
            }.getOrNull()
            return fromCookie ?: fromBody ?: throw IllegalStateException("The room did not return a session")
        }
    }

    private fun openSocket() {
        val url = roomBase!!.resolve("api/ws")!!.newBuilder().addQueryParameter("token", token).build()
        ws = http.newWebSocket(Request.Builder().url(url).build(), object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                if (webSocket === ws) handle(text)
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                if (webSocket === ws) dropped()
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                if (webSocket === ws) dropped()
            }
        })
    }

    // dropped reconnects a few times before giving up
    private fun dropped() {
        if (closed) return
        ws = null
        closePeer()
        if (retries >= 5) {
            fail("Lost the connection to the room")
            return
        }
        retries++
        _state.update { it.copy(phase = Phase.Reconnecting) }
        job = scope.launch(Dispatchers.IO) {
            delay(1500L * retries)
            if (closed) return@launch
            try {
                token = login()
                openSocket()
            } catch (e: Exception) {
                dropped()
            }
        }
    }

    private fun fail(message: String) {
        _state.update { it.copy(phase = Phase.Ended, error = message, video = null) }
    }

    private fun send(event: String, payload: JsonElement? = null) {
        val msg = buildJsonObject {
            put("event", event)
            if (payload != null) put("payload", payload)
        }
        ws?.send(msg.toString())
    }

    // websocket events

    private fun handle(text: String) {
        val msg = runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull() ?: return
        val event = msg["event"]?.jsonPrimitive?.contentOrNull ?: return
        val p = msg["payload"] as? JsonObject ?: JsonObject(emptyMap())

        when (event) {
            "system/init" -> {
                retries = 0
                val sessions = (p["sessions"] as? JsonObject)?.mapValues { (id, v) -> member(id, v.jsonObject) } ?: emptyMap()
                val screen = p["screen_size"] as? JsonObject
                val host = p["control_host"] as? JsonObject
                _state.update {
                    it.copy(
                        myId = p.str("session_id"),
                        members = sessions,
                        hostId = if (host?.bool("has_host") == true) host.str("host_id") else null,
                        screenWidth = screen?.int("width") ?: it.screenWidth,
                        screenHeight = screen?.int("height") ?: it.screenHeight,
                        touchSupported = p.bool("touch_events") == true,
                    )
                }
                send("signal/request", buildJsonObject {
                    put("video", JsonObject(emptyMap()))
                    put("audio", JsonObject(emptyMap()))
                })
            }

            "system/heartbeat" -> send("client/heartbeat")

            "system/disconnect" -> {
                closed = true
                fail(p.str("message")?.let { "Disconnected: $it" } ?: "Disconnected by the room")
            }

            "signal/provide", "signal/offer" -> offer(p)

            "signal/candidate" -> candidate(p)

            "session/created" -> {
                val id = p.str("id") ?: return
                _state.update { it.copy(members = it.members + (id to member(id, p))) }
            }

            "session/deleted" -> {
                val id = p.str("id") ?: return
                _state.update { it.copy(members = it.members - id) }
            }

            "session/profile" -> {
                val id = p.str("id") ?: return
                _state.update { s ->
                    val m = s.members[id] ?: return@update s
                    s.copy(members = s.members + (id to m.copy(name = p.str("name") ?: m.name, isAdmin = p.bool("is_admin") ?: m.isAdmin)))
                }
            }

            "session/state" -> {
                val id = p.str("id") ?: return
                val connected = p.bool("is_connected") ?: return
                _state.update { s ->
                    val m = s.members[id] ?: return@update s
                    if (m.connected == connected) return@update s
                    val line = if (id == s.myId) null else system(if (connected) "${m.name} joined" else "${m.name} left")
                    s.copy(members = s.members + (id to m.copy(connected = connected)), chat = s.chat.plusLine(line))
                }
            }

            "control/host" -> {
                val hostId = if (p.bool("has_host") == true) p.str("host_id") else null
                _state.update { s ->
                    if (s.hostId == hostId) return@update s
                    val line = when {
                        hostId == null -> system("Control released")
                        hostId == s.myId -> system("You have control")
                        else -> system("${s.members[hostId]?.name ?: "Someone"} has control")
                    }
                    s.copy(hostId = hostId, chat = s.chat.plusLine(line))
                }
            }

            "screen/updated" -> _state.update {
                it.copy(screenWidth = p.int("width") ?: it.screenWidth, screenHeight = p.int("height") ?: it.screenHeight)
            }

            "chat/message" -> {
                val text = (p["content"] as? JsonObject)?.str("text") ?: return
                val from = p.str("id")
                _state.update { s ->
                    val line = ChatLine(keys.incrementAndGet(), s.members[from]?.name ?: "Someone", text, false, from == s.myId)
                    s.copy(chat = s.chat.plusLine(line))
                }
            }
        }
    }

    private fun member(id: String, data: JsonObject): Member {
        val profile = data["profile"] as? JsonObject
        val state = data["state"] as? JsonObject
        return Member(
            id = id,
            name = profile?.str("name") ?: "Someone",
            isAdmin = profile?.bool("is_admin") == true,
            connected = state?.bool("is_connected") ?: true,
        )
    }

    private fun system(text: String) = ChatLine(keys.incrementAndGet(), "", text, true, false)

    private fun List<ChatLine>.plusLine(line: ChatLine?): List<ChatLine> =
        if (line == null) this else (this + line).takeLast(300)

    // WebRTC

    private fun offer(p: JsonObject) {
        val sdp = p.str("sdp") ?: return
        val peer = pc ?: createPeer(p["iceservers"] as? JsonArray) ?: return
        remoteSet = false
        peer.setRemoteDescription(object : Sdp() {
            override fun onSetSuccess() {
                remoteSet = true
                synchronized(pending) {
                    pending.forEach { peer.addIceCandidate(it) }
                    pending.clear()
                }
                answer(peer)
            }
        }, SessionDescription(SessionDescription.Type.OFFER, sdp))
    }

    private fun answer(peer: PeerConnection) {
        peer.createAnswer(object : Sdp() {
            override fun onCreateSuccess(desc: SessionDescription) {
                peer.setLocalDescription(object : Sdp() {
                    override fun onSetSuccess() {
                        send("signal/answer", buildJsonObject { put("sdp", desc.description) })
                    }
                }, desc)
            }
        }, MediaConstraints())
    }

    private fun candidate(p: JsonObject) {
        val c = IceCandidate(p.str("sdpMid") ?: "0", p.int("sdpMLineIndex") ?: 0, p.str("candidate") ?: return)
        val peer = pc
        if (peer != null && remoteSet) {
            peer.addIceCandidate(c)
        } else {
            synchronized(pending) { pending += c }
        }
    }

    private fun createPeer(servers: JsonArray?): PeerConnection? {
        val ice = servers?.mapNotNull { s ->
            val o = s as? JsonObject ?: return@mapNotNull null
            val urls = when (val u = o["urls"]) {
                is JsonArray -> u.mapNotNull { (it as? JsonPrimitive)?.contentOrNull }
                is JsonPrimitive -> listOfNotNull(u.contentOrNull)
                else -> emptyList()
            }
            if (urls.isEmpty()) return@mapNotNull null
            PeerConnection.IceServer.builder(urls)
                .setUsername(o.str("username") ?: "")
                .setPassword(o.str("credential") ?: "")
                .createIceServer()
        } ?: emptyList()

        val config = PeerConnection.RTCConfiguration(ice).apply {
            sdpSemantics = PeerConnection.SdpSemantics.UNIFIED_PLAN
            bundlePolicy = PeerConnection.BundlePolicy.MAXBUNDLE
            rtcpMuxPolicy = PeerConnection.RtcpMuxPolicy.REQUIRE
        }

        val peer = WebRtc.factory.createPeerConnection(config, object : PeerConnection.Observer {
            override fun onIceCandidate(c: IceCandidate) {
                send("signal/candidate", buildJsonObject {
                    put("candidate", c.sdp)
                    put("sdpMid", c.sdpMid)
                    put("sdpMLineIndex", c.sdpMLineIndex)
                })
            }

            override fun onTrack(transceiver: RtpTransceiver) {
                when (val track = transceiver.receiver.track()) {
                    is VideoTrack -> _state.update { it.copy(video = track, phase = Phase.Live) }
                    is AudioTrack -> {
                        audio = track
                        track.setEnabled(!_state.value.muted)
                    }
                }
            }

            override fun onDataChannel(channel: DataChannel) {
                dc = channel
            }

            override fun onIceConnectionChange(s: PeerConnection.IceConnectionState) {
                if (s == PeerConnection.IceConnectionState.FAILED) {
                    // media path broke, renegotiate over the websocket
                    scope.launch(Dispatchers.IO) { dropped() }
                }
            }

            override fun onSignalingChange(s: PeerConnection.SignalingState) {}
            override fun onIceConnectionReceivingChange(receiving: Boolean) {}
            override fun onIceGatheringChange(s: PeerConnection.IceGatheringState) {}
            override fun onIceCandidatesRemoved(candidates: Array<out IceCandidate>) {}
            override fun onAddStream(stream: MediaStream) {}
            override fun onRemoveStream(stream: MediaStream) {}
            override fun onRenegotiationNeeded() {}
        })
        pc = peer
        return peer
    }

    private fun closePeer() {
        _state.update { it.copy(video = null) }
        runCatching { dc?.close() }
        runCatching { pc?.close() }
        dc = null
        pc = null
        audio = null
        remoteSet = false
        synchronized(pending) { pending.clear() }
    }

    // controls

    fun sendChat(text: String) {
        val t = text.trim()
        if (t.isEmpty()) return
        send("chat/message", buildJsonObject { put("text", t) })
    }

    fun requestControl() = send("control/request")

    fun releaseControl() = send("control/release")

    fun setMuted(muted: Boolean) {
        audio?.setEnabled(!muted)
        _state.update { it.copy(muted = muted) }
    }

    // input over the data channel, big endian: op (u8), length (u16), payload

    private fun input(op: Int, payload: ByteBuffer.() -> Unit, length: Int) {
        val channel = dc ?: return
        if (channel.state() != DataChannel.State.OPEN) return
        val buf = ByteBuffer.allocate(3 + length)
        buf.put(op.toByte())
        buf.putShort(length.toShort())
        buf.payload()
        buf.flip()
        channel.send(DataChannel.Buffer(buf, true))
    }

    fun move(x: Int, y: Int) = input(0x01, { putShort(x.toShort()); putShort(y.toShort()) }, 4)

    // scroll in wheel steps, positive y scrolls up
    fun scroll(dx: Int, dy: Int) = input(0x02, { putShort(dx.toShort()); putShort(dy.toShort()); put(0) }, 5)

    fun keyDown(keysym: Int) = input(0x03, { putInt(keysym) }, 4)

    fun keyUp(keysym: Int) = input(0x04, { putInt(keysym) }, 4)

    fun key(keysym: Int, ctrl: Boolean = false) {
        if (ctrl) keyDown(Keys.CONTROL)
        keyDown(keysym)
        keyUp(keysym)
        if (ctrl) keyUp(Keys.CONTROL)
    }

    fun buttonDown(button: Int) = input(0x05, { putInt(button) }, 4)

    fun buttonUp(button: Int) = input(0x06, { putInt(button) }, 4)

    fun click(x: Int, y: Int, button: Int = Buttons.LEFT) {
        move(x, y)
        buttonDown(button)
        buttonUp(button)
    }

    private fun touch(op: Int, id: Int, x: Int, y: Int, pressure: Int) =
        input(op, { putInt(id); putInt(x); putInt(y); put(pressure.coerceIn(0, 255).toByte()) }, 13)

    fun touchBegin(id: Int, x: Int, y: Int, pressure: Int = 255) = touch(0x08, id, x, y, pressure)

    fun touchUpdate(id: Int, x: Int, y: Int, pressure: Int = 255) = touch(0x09, id, x, y, pressure)

    fun touchEnd(id: Int, x: Int, y: Int) = touch(0x0a, id, x, y, 0)

    companion object {
        private val JSON = "application/json".toMediaType()
    }
}

private fun JsonObject.str(key: String): String? = (this[key] as? JsonPrimitive)?.contentOrNull
private fun JsonObject.int(key: String): Int? = (this[key] as? JsonPrimitive)?.intOrNull
private fun JsonObject.bool(key: String): Boolean? = (this[key] as? JsonPrimitive)?.booleanOrNull

private open class Sdp : SdpObserver {
    override fun onCreateSuccess(desc: SessionDescription) {}
    override fun onSetSuccess() {}
    override fun onCreateFailure(error: String?) {}
    override fun onSetFailure(error: String?) {}
}
