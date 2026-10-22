package app.rooms.data

import android.content.Context
import okhttp3.Cookie
import okhttp3.CookieJar
import okhttp3.HttpUrl
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull

// PersistentCookieJar keeps the site session across app restarts.
class PersistentCookieJar(context: Context) : CookieJar {
    private val sp = context.getSharedPreferences("cookies", Context.MODE_PRIVATE)
    private val cookies = mutableListOf<Cookie>()

    init {
        sp.getStringSet("all", emptySet())?.forEach { line ->
            val origin = line.substringBefore('\t').toHttpUrlOrNull() ?: return@forEach
            Cookie.parse(origin, line.substringAfter('\t'))?.let { cookies += it }
        }
        prune()
    }

    @Synchronized
    override fun saveFromResponse(url: HttpUrl, cookies: List<Cookie>) {
        for (c in cookies) {
            this.cookies.removeAll { it.name == c.name && it.domain == c.domain && it.path == c.path }
            this.cookies += c
        }
        prune()
        persist()
    }

    @Synchronized
    override fun loadForRequest(url: HttpUrl): List<Cookie> {
        prune()
        return cookies.filter { it.matches(url) }
    }

    @Synchronized
    fun clear() {
        cookies.clear()
        persist()
    }

    private fun prune() {
        val now = System.currentTimeMillis()
        cookies.removeAll { it.expiresAt < now }
    }

    private fun persist() {
        val lines = cookies.filter { it.persistent }.map { c ->
            val scheme = if (c.secure) "https" else "http"
            "$scheme://${c.domain}${c.path}\t$c"
        }.toSet()
        sp.edit().putStringSet("all", lines).apply()
    }
}
