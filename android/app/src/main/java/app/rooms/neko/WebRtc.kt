package app.rooms.neko

import android.content.Context
import android.media.AudioAttributes
import org.webrtc.DefaultVideoDecoderFactory
import org.webrtc.DefaultVideoEncoderFactory
import org.webrtc.EglBase
import org.webrtc.PeerConnectionFactory
import org.webrtc.audio.JavaAudioDeviceModule

// WebRtc holds the process wide WebRTC objects.
object WebRtc {
    lateinit var egl: EglBase
        private set
    lateinit var factory: PeerConnectionFactory
        private set
    private var ready = false

    @Synchronized
    fun init(context: Context) {
        if (ready) return
        val app = context.applicationContext
        PeerConnectionFactory.initialize(
            PeerConnectionFactory.InitializationOptions.builder(app).createInitializationOptions()
        )
        egl = EglBase.create()

        // play room audio like media (speaker, media volume), not like a call
        val audio = JavaAudioDeviceModule.builder(app)
            .setUseStereoOutput(true)
            .setAudioAttributes(
                AudioAttributes.Builder()
                    .setUsage(AudioAttributes.USAGE_MEDIA)
                    .setContentType(AudioAttributes.CONTENT_TYPE_MOVIE)
                    .build()
            )
            .createAudioDeviceModule()

        factory = PeerConnectionFactory.builder()
            .setAudioDeviceModule(audio)
            .setVideoDecoderFactory(DefaultVideoDecoderFactory(egl.eglBaseContext))
            .setVideoEncoderFactory(DefaultVideoEncoderFactory(egl.eglBaseContext, true, true))
            .createPeerConnectionFactory()
        ready = true
    }
}
