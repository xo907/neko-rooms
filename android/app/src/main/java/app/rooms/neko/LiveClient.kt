package app.rooms.neko

import app.rooms.data.json
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

@Serializable
data class QueueItem(
    val id: Int = 0,
    val url: String = "",
    val title: String = "",
    val by: String = "",
    @SerialName("by_id") val byId: String = "",
    val voters: List<String> = emptyList(),
) {
    val host: String get() = runCatching { java.net.URI(url).host.removePrefix("www.") }.getOrNull() ?: url
}

@Serializable
data class Hand(val id: String = "", val name: String = "")

@Serializable
data class LiveMe(val id: String = "", val name: String = "", val manager: Boolean = false)

data class LiveState(
    val connected: Boolean = false,
    val me: LiveMe? = null,
    val queue: List<QueueItem> = emptyList(),
    val hands: List<Hand> = emptyList(),
    val playing: String = "",
) {
    val handUp: Boolean get() = me != null && hands.any { it.id == me.id }
}

data class Reaction(val key: Long, val emoji: String, val name: String)

// LiveClient is the room's live channel on the neko-rooms server:
// reactions, raised hands, the up next queue and what is playing.
class LiveClient(http: OkHttpClient, private val api: HttpUrl, private val room: String, private val ticket: String, private val name: String) {
    companion object {
        val Reactions = listOf("👍", "😂", "😮", "❤️", "🔥", "👏", "🎉", "😢")
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val http = http.newBuilder().readTimeout(0, TimeUnit.MILLISECONDS).build()
    private val base = api.resolve("public/live/${java.net.URLEncoder.encode(room, "UTF-8").replace("+", "%20")}/")!!

    private val _state = MutableStateFlow(LiveState())
    val state: StateFlow<LiveState> = _state

    private val _reactions = MutableSharedFlow<Reaction>(extraBufferCapacity = 32)
    val reactions: SharedFlow<Reaction> = _reactions

    private val _errors = MutableSharedFlow<String>(extraBufferCapacity = 4)
    val errors: SharedFlow<String> = _errors

    private var counter = 0L

    fun start() {
        scope.launch {
            var backoff = 1000L
            while (isActive) {
                runCatching { listen() }
                _state.update { it.copy(connected = false) }
                delay(backoff)
                backoff = (backoff * 2).coerceAtMost(15_000)
            }
        }
    }

    fun stop() {
        scope.cancel()
    }

    private fun listen() {
        val url = base.resolve("events")!!.newBuilder()
            .addQueryParameter("t", ticket)
            .addQueryParameter("name", name)
            .build()
        http.newCall(Request.Builder().url(url).header("Accept", "text/event-stream").build()).execute().use { res ->
            if (!res.isSuccessful) throw IllegalStateException("live channel ${res.code}")
            _state.update { it.copy(connected = true) }
            val source = res.body!!.source()
            while (!source.exhausted()) {
                val line = source.readUtf8Line() ?: break
                if (line.startsWith("data: ")) handle(line.removePrefix("data: "))
            }
        }
    }

    private fun handle(text: String) {
        val ev = runCatching { json.parseToJsonElement(text).jsonObject }.getOrNull() ?: return
        fun str(key: String) = ev[key]?.jsonPrimitive?.contentOrNull
        fun queue() = ev["queue"]?.let { json.decodeFromJsonElement(ListSerializer(QueueItem.serializer()), it) }
        fun hands() = ev["hands"]?.let { json.decodeFromJsonElement(ListSerializer(Hand.serializer()), it) }
        when (str("type")) {
            "init" -> _state.update {
                it.copy(
                    me = ev["you"]?.let { y -> json.decodeFromJsonElement(LiveMe.serializer(), y) },
                    queue = queue() ?: emptyList(),
                    hands = hands() ?: emptyList(),
                    playing = str("playing") ?: "",
                )
            }
            "queue" -> _state.update { it.copy(queue = queue() ?: emptyList()) }
            "hands" -> _state.update { it.copy(hands = hands() ?: emptyList()) }
            "playing" -> _state.update { it.copy(playing = str("title") ?: "") }
            "reaction" -> _reactions.tryEmit(Reaction(counter++, str("emoji") ?: return, str("name") ?: ""))
            "played" -> {
                val item = (ev["item"] as? JsonObject)?.let { json.decodeFromJsonElement(QueueItem.serializer(), it) }
                _errors.tryEmit("${str("name") ?: "Someone"} played ${item?.title?.ifBlank { null } ?: item?.host ?: "a link"}")
            }
        }
    }

    private fun post(path: String, extra: JsonObject = JsonObject(emptyMap())) {
        scope.launch {
            val body = buildJsonObject {
                put("t", ticket)
                put("name", name)
                extra.forEach { (k, v) -> put(k, v) }
            }
            val req = Request.Builder()
                .url(base.resolve(path)!!)
                .post(body.toString().toRequestBody("application/json".toMediaType()))
                .build()
            runCatching {
                http.newCall(req).execute().use { res ->
                    if (!res.isSuccessful) _errors.tryEmit(res.body?.string()?.trim()?.ifBlank { null } ?: "Error ${res.code}")
                }
            }.onFailure { _errors.tryEmit("No connection") }
        }
    }

    fun react(emoji: String) = post("react", buildJsonObject { put("emoji", emoji) })

    fun hand(up: Boolean, target: String? = null) = post("hand", buildJsonObject {
        put("up", up)
        if (target != null) put("target", target)
    })

    fun add(url: String) = post("queue", buildJsonObject { put("url", url) })

    fun vote(id: Int) = post("queue/$id/vote")

    fun play(id: Int) = post("queue/$id/play")

    fun remove(id: Int) = post("queue/$id/remove")
}
