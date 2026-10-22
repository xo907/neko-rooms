package app.rooms

import android.app.Application
import app.rooms.data.Api
import app.rooms.data.PersistentCookieJar
import app.rooms.data.Prefs
import coil.ImageLoader
import coil.ImageLoaderFactory
import okhttp3.OkHttpClient
import java.util.concurrent.TimeUnit

class WatchPartyApp : Application(), ImageLoaderFactory {
    lateinit var prefs: Prefs
        private set
    lateinit var cookies: PersistentCookieJar
        private set
    lateinit var http: OkHttpClient
        private set
    lateinit var api: Api
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        prefs = Prefs(this)
        cookies = PersistentCookieJar(this)
        http = OkHttpClient.Builder()
            .cookieJar(cookies)
            // required by the server's CSRF protection, like the site sends it
            .addInterceptor { chain ->
                chain.proceed(chain.request().newBuilder().header("X-Requested-With", "XMLHttpRequest").build())
            }
            .connectTimeout(15, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .pingInterval(20, TimeUnit.SECONDS)
            .build()
        api = Api(http).apply { base = prefs.server }
    }

    // thumbnails and avatars load with the same session as the API
    override fun newImageLoader(): ImageLoader =
        ImageLoader.Builder(this).okHttpClient(http).crossfade(true).build()

    companion object {
        lateinit var instance: WatchPartyApp
            private set
    }
}
