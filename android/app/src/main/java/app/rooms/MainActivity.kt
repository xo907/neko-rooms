package app.rooms

import android.app.PictureInPictureParams
import android.content.pm.PackageManager
import android.content.Intent
import android.media.AudioManager
import android.widget.Toast
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.mutableStateOf
import androidx.compose.ui.platform.LocalContext
import app.rooms.data.normalizeServer
import okhttp3.HttpUrl
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.util.Rational
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.ui.Modifier
import androidx.lifecycle.viewmodel.compose.viewModel
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.rememberNavController
import androidx.navigation.navArgument
import app.rooms.ui.AccountScreen
import app.rooms.ui.AppModel
import app.rooms.ui.FriendsScreen
import app.rooms.ui.HomeScreen
import app.rooms.ui.LoginScreen
import app.rooms.ui.PlayerScreen
import app.rooms.ui.RegisterScreen
import app.rooms.ui.RoomScreen
import app.rooms.ui.SetupScreen
import app.rooms.ui.WatchPartyTheme

class MainActivity : ComponentActivity() {
    // a room link the app was opened with, see App
    val link = mutableStateOf<Uri?>(null)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (savedInstanceState == null) link.value = intent?.data
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(android.graphics.Color.TRANSPARENT),
        )
        // volume keys change media volume, also while a room plays
        volumeControlStream = AudioManager.STREAM_MUSIC

        setContent {
            WatchPartyTheme {
                Box(Modifier.fillMaxSize().background(MaterialTheme.colorScheme.background)) {
                    App()
                }
            }
        }
    }

    // the app is already open: the link replaces what is on screen
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        intent.data?.let { link.value = it }
    }

    // set while a room plays, leaving the app then shrinks it into a
    // floating window instead of hiding it
    var pipRatio: Rational? = null
        set(value) {
            field = value
            updatePip()
        }

    private val pipSupported by lazy { packageManager.hasSystemFeature(PackageManager.FEATURE_PICTURE_IN_PICTURE) }

    private fun pipParams(): PictureInPictureParams {
        val b = PictureInPictureParams.Builder()
        pipRatio?.let { b.setAspectRatio(it) }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) b.setAutoEnterEnabled(pipRatio != null)
        return b.build()
    }

    private fun updatePip() {
        if (pipSupported) runCatching { setPictureInPictureParams(pipParams()) }
    }

    // Android 12 and later enter by themselves, see setAutoEnterEnabled
    override fun onUserLeaveHint() {
        super.onUserLeaveHint()
        if (pipSupported && pipRatio != null && Build.VERSION.SDK_INT < Build.VERSION_CODES.S) {
            runCatching { enterPictureInPictureMode(pipParams()) }
        }
    }
}

private fun enc(s: String) = Uri.encode(s)

private fun NavHostController.room(name: String, invite: String?) =
    navigate("room/${enc(name)}?invite=${enc(invite.orEmpty())}")

// RoomLink is a room on some server, from watchparty://room?server=&name=&invite=
private data class RoomLink(val server: HttpUrl, val name: String, val invite: String?)

private fun roomLink(uri: Uri): RoomLink? {
    if (uri.scheme != "watchparty" || uri.host != "room") return null
    val server = normalizeServer(uri.getQueryParameter("server") ?: return null) ?: return null
    val name = uri.getQueryParameter("name")?.takeIf { it.isNotBlank() } ?: return null
    return RoomLink(server, name, uri.getQueryParameter("invite")?.ifBlank { null })
}

@androidx.compose.runtime.Composable
private fun App() {
    val model: AppModel = viewModel()
    val nav = rememberNavController()
    val start = if (model.prefs.server == null) "setup" else "home"
    val activity = LocalContext.current as? MainActivity
    val context = LocalContext.current

    // a room opened from a site, on this server or another one
    val pending = activity?.link?.value
    LaunchedEffect(pending) {
        val uri = pending ?: return@LaunchedEffect
        activity?.link?.value = null
        val target = roomLink(uri) ?: return@LaunchedEffect
        if (model.prefs.server != target.server && model.useServer(target.server).isFailure) {
            Toast.makeText(context, "Could not reach ${target.server.host}", Toast.LENGTH_LONG).show()
            return@LaunchedEffect
        }
        nav.navigate("home") { popUpTo(0) }
        nav.room(target.name, target.invite)
    }

    NavHost(navController = nav, startDestination = start) {
        composable("setup") {
            SetupScreen(model) { room, invite ->
                nav.navigate("home") { popUpTo(0) }
                if (room != null) nav.room(room, invite)
            }
        }

        composable("home") {
            HomeScreen(
                model = model,
                onRoom = { name, invite -> nav.room(name, invite) },
                onAccount = { nav.navigate("account") },
                onSignIn = { nav.navigate("login") },
                onChangeServer = { nav.navigate("setup") },
            )
        }

        composable(
            "room/{name}?invite={invite}",
            arguments = listOf(
                navArgument("name") { type = NavType.StringType },
                navArgument("invite") { type = NavType.StringType; defaultValue = "" },
            ),
        ) { entry ->
            val name = entry.arguments?.getString("name").orEmpty()
            val invite = entry.arguments?.getString("invite")?.ifEmpty { null }
            RoomScreen(
                model = model,
                name = name,
                invite = invite,
                onBack = { nav.popBackStack() },
                onJoin = { nick ->
                    nav.navigate("watch/${enc(name)}?invite=${enc(invite.orEmpty())}&nick=${enc(nick.orEmpty())}")
                },
                onSignIn = { nav.navigate("login") },
            )
        }

        composable(
            "watch/{name}?invite={invite}&nick={nick}",
            arguments = listOf(
                navArgument("name") { type = NavType.StringType },
                navArgument("invite") { type = NavType.StringType; defaultValue = "" },
                navArgument("nick") { type = NavType.StringType; defaultValue = "" },
            ),
        ) { entry ->
            PlayerScreen(
                model = model,
                name = entry.arguments?.getString("name").orEmpty(),
                invite = entry.arguments?.getString("invite")?.ifEmpty { null },
                nickname = entry.arguments?.getString("nick")?.ifEmpty { null },
                onBack = { nav.popBackStack() },
            )
        }

        composable("login") {
            LoginScreen(
                model = model,
                onDone = { nav.popBackStack() },
                onRegister = { nav.navigate("register") { popUpTo("login") { inclusive = true } } },
                onBack = { nav.popBackStack() },
            )
        }

        composable("register") {
            RegisterScreen(model = model, onDone = { nav.popBackStack() }, onBack = { nav.popBackStack() })
        }

        composable("account") {
            AccountScreen(
                model = model,
                onBack = { nav.popBackStack() },
                onFriends = { nav.navigate("friends") },
                onChangeServer = { nav.navigate("setup") },
            )
        }

        composable("friends") {
            FriendsScreen(model = model, onBack = { nav.popBackStack() })
        }
    }
}
