package app.rooms.data

import android.content.Context
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

class Prefs(context: Context) {
    private val sp = context.getSharedPreferences("watchparty", Context.MODE_PRIVATE)

    var server: HttpUrl?
        get() = sp.getString("server", null)?.toHttpUrlOrNull()
        set(value) = sp.edit().putString("server", value?.toString()).apply()

    // nickname used when joining rooms without an account
    var nickname: String
        get() = sp.getString("nickname", "") ?: ""
        set(value) = sp.edit().putString("nickname", value).apply()

    // native touch input instead of mouse emulation, when the room supports it
    var touchInput: Boolean
        get() = sp.getBoolean("touch_input", true)
        set(value) = sp.edit().putBoolean("touch_input", value).apply()
}

private val ipLike = Regex("""^\d{1,3}(\.\d{1,3}){3}$""")

// normalizeServer turns what people type into a base url ending with a slash:
// "rooms.example.com" -> https, IPs, localhost, tailnet names or ports -> http
fun normalizeServer(input: String): HttpUrl? {
    var s = input.trim().substringBefore('#')
    if (s.isEmpty()) return null
    if (!s.startsWith("http://", ignoreCase = true) && !s.startsWith("https://", ignoreCase = true)) {
        val hostPort = s.substringBefore('/')
        val host = hostPort.substringBefore(':')
        val plain = ipLike.matches(host) || host == "localhost" || host.endsWith(".ts.net") ||
            !host.contains('.') || hostPort.contains(':')
        s = (if (plain) "http://" else "https://") + s
    }
    if (!s.endsWith("/")) s += "/"
    return s.toHttpUrlOrNull()
}

data class InviteLink(val server: HttpUrl?, val room: String, val invite: String?)

// parseInviteLink reads links copied from the site, e.g.
// https://rooms.example.com/r/movie-night?invite=abc, or the older
// https://rooms.example.com/#/r/movie-night?invite=abc
fun parseInviteLink(input: String): InviteLink? {
    val text = input.trim()
    val hash = text.indexOf("#/r/")
    val at = if (hash >= 0) hash else text.indexOf("/r/", text.indexOf("://").let { if (it < 0) 0 else it + 3 })
    if (at < 0) return null
    val marker = if (hash >= 0) "#/r/" else "/r/"
    val rest = text.substring(at + marker.length)
    val room = rest.substringBefore('?').substringBefore('/').substringBefore('#').trim()
    if (room.isEmpty()) return null
    val invite = rest.substringAfter("invite=", "").substringBefore('&').substringBefore('#').ifEmpty { null }
    val server = normalizeServer(text.substring(0, if (hash >= 0) at else at + 1))
    return InviteLink(server, java.net.URLDecoder.decode(room, "UTF-8"), invite?.let { java.net.URLDecoder.decode(it, "UTF-8") })
}
