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
data class CaptionLanguage(val code: String = "", val name: String = "")

@Serializable
data class RewindSegment(val n: Int = 0, val start: Long = 0, val end: Long = 0)

// RewindIndex is what the server kept of the room, times in ms on its
// recording timeline
@Serializable
data class RewindIndex(val now: Long = 0, val keep: Long = 0, val recording: Boolean = false, val segments: List<RewindSegment> = emptyList()) {
    val first: Long get() = segments.firstOrNull()?.start ?: 0
    val last: Long get() = segments.lastOrNull()?.end ?: 0
    val available: Boolean get() = segments.isNotEmpty() && last - first >= 15_000
}

// streamPositions is where each segment starts in the rewind stream, the
// same way the server joins them: gaps over a second are closed
fun streamPositions(segs: List<RewindSegment>): LongArray {
    val pos = LongArray(segs.size)
    for (i in 1 until segs.size) {
        val prev = segs[i - 1]
        val s = segs[i]
        pos[i] = pos[i - 1] + if (s.start - prev.end <= 1000) s.start - prev.start else prev.end - prev.start
    }
    return pos
}

@Serializable
data class LiveMe(val id: String = "", val name: String = "", val manager: Boolean = false)

data class LiveState(
    val connected: Boolean = false,
    val me: LiveMe? = null,
    val queue: List<QueueItem> = emptyList(),
    val hands: List<Hand> = emptyList(),
    val playing: String = "",
    val captionLangs: List<CaptionLanguage> = emptyList(),
    val captions: String = "", // language picked, empty for none
) {
    val handUp: Boolean get() = me != null && hands.any { it.id == me.id }
}

data class Reaction(val key: Long, val emoji: String, val name: String)

// LiveClient is the room's live channel on the neko-rooms server:
// reactions, raised hands, the up next queue and what is playing.
class LiveClient(http: OkHttpClient, private val api: HttpUrl, private val room: String, private val ticket: String, private val name: String, captions: String = "") {
    companion object {
        // all of them are neko emotes too, so browsers show them the same
        val Reactions = listOf("👍", "🔥", "👏", "🎉", "😍", "😮", "😭", "💯")
    }

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val http = http.newBuilder().readTimeout(0, TimeUnit.MILLISECONDS).build()
    private val base = api.resolve("public/live/${java.net.URLEncoder.encode(room, "UTF-8").replace("+", "%20")}/")!!

    private val plain = http
    private val _state = MutableStateFlow(LiveState(captions = captions))
    val state: StateFlow<LiveState> = _state

    private val _reactions = MutableSharedFlow<Reaction>(extraBufferCapacity = 32)
    val reactions: SharedFlow<Reaction> = _reactions

    private val _captions = MutableSharedFlow<String>(extraBufferCapacity = 16)
    val captionLines: SharedFlow<String> = _captions

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
            }.also { joinCaptions() }
            "queue" -> _state.update { it.copy(queue = queue() ?: emptyList()) }
            "hands" -> _state.update { it.copy(hands = hands() ?: emptyList()) }
            "playing" -> _state.update { it.copy(playing = str("title") ?: "") }
            "caption" -> if (str("lang") == _state.value.captions) _captions.tryEmit(str("text") ?: return)
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

    private fun get(path: String, query: Map<String, String> = emptyMap()): String? {
        val url = base.resolve(path)!!.newBuilder().addQueryParameter("t", ticket)
        query.forEach { (k, v) -> url.addQueryParameter(k, v) }
        return runCatching {
            plain.newCall(Request.Builder().url(url.build()).build()).execute().use { res ->
                if (res.isSuccessful) res.body?.string() else null
            }
        }.getOrNull()
    }

    // the subtitle languages, and the one picked before, after every connect
    private fun joinCaptions() {
        scope.launch {
            val langs = get("captions")?.let {
                runCatching {
                    json.decodeFromJsonElement(ListSerializer(CaptionLanguage.serializer()), json.parseToJsonElement(it).jsonObject["languages"]!!)
                }.getOrNull()
            } ?: emptyList()
            val picked = _state.value.captions
            val keep = picked.isNotEmpty() && langs.any { it.code == picked }
            _state.update { it.copy(captionLangs = langs, captions = if (keep || langs.isEmpty()) picked else "") }
            if (keep) post("captions", buildJsonObject { put("lang", picked) })
        }
    }

    // setCaptions picks the subtitle language, "" turns them off
    fun setCaptions(lang: String) {
        _state.update { it.copy(captions = lang) }
        post("captions", buildJsonObject { put("lang", lang) })
    }

    // rewindIndex is what the server kept of the room, null when nothing
    suspend fun rewindIndex(): RewindIndex? = kotlinx.coroutines.withContext(Dispatchers.IO) {
        get("rewind")?.let { runCatching { json.decodeFromString(RewindIndex.serializer(), it) }.getOrNull() }
    }

    // rewindStream is the recording from segment n on, as one WebM
    fun rewindStream(n: Int): String = base.resolve("rewind/stream")!!.newBuilder()
        .addQueryParameter("from", n.toString())
        .addQueryParameter("t", ticket)
        .build().toString()

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
