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
// https://rooms.example.com/#/r/movie-night?invite=abc
fun parseInviteLink(input: String): InviteLink? {
    val text = input.trim()
    val fragment = text.substringAfter("#/r/", "")
    if (fragment.isEmpty()) return null
    val room = fragment.substringBefore('?').substringBefore('/').trim()
    if (room.isEmpty()) return null
    val invite = fragment.substringAfter("invite=", "").substringBefore('&').ifEmpty { null }
    val server = normalizeServer(text.substringBefore('#'))
    return InviteLink(server, java.net.URLDecoder.decode(room, "UTF-8"), invite)
}
