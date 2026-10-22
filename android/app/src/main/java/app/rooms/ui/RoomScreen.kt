package app.rooms.ui

import android.content.Intent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.PlayArrow
import androidx.compose.material.icons.outlined.Share
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import app.rooms.data.ApiException
import app.rooms.data.Room
import kotlinx.coroutines.delay

@OptIn(ExperimentalMaterial3Api::class, ExperimentalLayoutApi::class)
@Composable
fun RoomScreen(
    model: AppModel,
    name: String,
    invite: String?,
    onBack: () -> Unit,
    onJoin: (nickname: String?) -> Unit,
    onSignIn: () -> Unit,
) {
    val status by model.status.collectAsState()
    val context = LocalContext.current
    var room by remember { mutableStateOf<Room?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var forbidden by remember { mutableStateOf(false) }
    var bust by remember { mutableLongStateOf(System.currentTimeMillis() / 15000) }
    var askNickname by remember { mutableStateOf(false) }

    LaunchedEffect(name, invite, status?.user?.id) {
        while (true) {
            runCatching { model.api.room(name, invite) }
                .onSuccess { room = it; error = null }
                .onFailure {
                    forbidden = (it as? ApiException)?.code == 403
                    error = if (forbidden) "This room is private" else "Room not found"
                }
            bust = System.currentTimeMillis() / 15000
            delay(15_000)
        }
    }

    fun join() {
        when {
            status?.user != null -> onJoin(null)
            status?.policy?.roomsRequireLogin == true -> onSignIn()
            else -> askNickname = true
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(room?.displayTitle ?: "", maxLines = 1) },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Back") } },
                actions = {
                    if (room != null) {
                        IconButton(onClick = {
                            val link = model.api.roomLink(name, invite)
                            context.startActivity(
                                Intent.createChooser(
                                    Intent(Intent.ACTION_SEND).setType("text/plain").putExtra(Intent.EXTRA_TEXT, link),
                                    "Share room",
                                )
                            )
                        }) { Icon(Icons.Outlined.Share, "Share") }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background),
            )
        },
    ) { padding ->
        val r = room
        when {
            r == null && error == null -> Centered(Modifier.padding(padding)) { CircularProgressIndicator(strokeWidth = 2.dp) }
            r == null -> Centered(Modifier.padding(padding)) {
                Column(horizontalAlignment = Alignment.CenterHorizontally, modifier = Modifier.padding(32.dp)) {
                    Text(error ?: "", style = MaterialTheme.typography.titleMedium)
                    Spacer(Modifier.height(8.dp))
                    Text(
                        if (forbidden) "Ask the owner for an invite link, or sign in if you are one of their friends." else "It may have been closed.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        textAlign = TextAlign.Center,
                    )
                    if (forbidden && status?.user == null) {
                        Spacer(Modifier.height(16.dp))
                        Button(onClick = onSignIn) { Text("Sign in") }
                    }
                }
            }
            else -> Column(
                Modifier
                    .padding(padding)
                    .fillMaxSize()
                    .verticalScroll(rememberScrollState())
                    .padding(horizontal = 16.dp)
                    .navigationBarsPadding(),
            ) {
                Box {
                    RoomThumb(r, model.api.thumbnailUrl(r.name, bust, invite))
                    if (r.canJoin) {
                        Button(
                            onClick = { join() },
                            enabled = r.running,
                            shape = RoundedCornerShape(50),
                            contentPadding = ButtonDefaults.ButtonWithIconContentPadding,
                            modifier = Modifier.align(Alignment.Center),
                        ) {
                            Icon(Icons.Outlined.PlayArrow, null, Modifier.size(20.dp))
                            Spacer(Modifier.width(6.dp))
                            Text(if (r.running) "Join room" else "Not running")
                        }
                    }
                }

                Spacer(Modifier.height(18.dp))
                Text(r.displayTitle, style = MaterialTheme.typography.headlineMedium)
                Spacer(Modifier.height(10.dp))
                FlowRow(horizontalArrangement = Arrangement.spacedBy(6.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Tag(if (r.ready) "Live" else if (r.paused) "Paused" else "Offline", highlight = r.ready)
                    Tag(r.visibility.replaceFirstChar { it.uppercase() })
                    if (r.category.isNotBlank()) Tag(r.category)
                    if (r.featured) Tag("Featured")
                }
                r.owner?.let { owner ->
                    Spacer(Modifier.height(14.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Avatar(owner.name, model.api.avatarUrl(owner.id, owner.avatar), 22.dp)
                        Spacer(Modifier.width(8.dp))
                        Text("Hosted by ", color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.bodyMedium)
                        Text(owner.name, style = MaterialTheme.typography.titleSmall)
                    }
                }
                if (r.description.isNotBlank()) {
                    Spacer(Modifier.height(16.dp))
                    HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
                    Spacer(Modifier.height(16.dp))
                    Text(r.description, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }

                Spacer(Modifier.height(24.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text("In this room", style = MaterialTheme.typography.titleSmall)
                    Spacer(Modifier.weight(1f))
                    Text("${r.viewers}", color = MaterialTheme.colorScheme.onSurfaceVariant, style = MaterialTheme.typography.labelMedium)
                }
                Spacer(Modifier.height(10.dp))
                if (r.members.isEmpty()) {
                    Text(
                        if (r.ready) "Nobody is here yet. Be the first!" else "The room is not running right now.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodyMedium,
                    )
                } else {
                    r.members.forEach { m ->
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(vertical = 6.dp)) {
                            Avatar(m, r.avatars[m]?.let { model.api.apiUrl(it) }, 30.dp)
                            Spacer(Modifier.width(12.dp))
                            Text(m, style = MaterialTheme.typography.bodyLarge)
                            if (m in r.friendsInside) {
                                Spacer(Modifier.width(8.dp))
                                Tag("Friend", highlight = true)
                            }
                        }
                    }
                }
                Spacer(Modifier.height(32.dp))
            }
        }
    }

    if (askNickname) {
        NicknameDialog(
            initial = model.prefs.nickname,
            onDismiss = { askNickname = false },
            onConfirm = { nick ->
                askNickname = false
                model.prefs.nickname = nick
                onJoin(nick)
            },
        )
    }
}

@Composable
fun Tag(text: String, highlight: Boolean = false) {
    val c = MaterialTheme.colorScheme
    Text(
        text,
        style = MaterialTheme.typography.labelMedium,
        color = if (highlight) Live else c.onSurfaceVariant,
        modifier = Modifier
            .background(if (highlight) Live.copy(alpha = 0.12f) else c.surfaceContainerHigh, RoundedCornerShape(50))
            .padding(horizontal = 10.dp, vertical = 4.dp),
    )
}
