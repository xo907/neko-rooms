package app.rooms

import android.media.AudioManager
import android.net.Uri
import android.os.Bundle
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
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
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
}

private fun enc(s: String) = Uri.encode(s)

private fun NavHostController.room(name: String, invite: String?) =
    navigate("room/${enc(name)}?invite=${enc(invite.orEmpty())}")

@androidx.compose.runtime.Composable
private fun App() {
    val model: AppModel = viewModel()
    val nav = rememberNavController()
    val start = if (model.prefs.server == null) "setup" else "home"

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
