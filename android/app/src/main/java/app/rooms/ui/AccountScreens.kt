package app.rooms.ui

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.graphics.ImageDecoder
import android.net.Uri
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.automirrored.outlined.KeyboardArrowRight
import androidx.compose.material.icons.outlined.Check
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.PersonAdd
import androidx.compose.material.icons.outlined.PhotoCamera
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.Badge
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.ListItem
import androidx.compose.material3.ListItemDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import app.rooms.data.Friend
import app.rooms.data.PublicUser
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.ByteArrayOutputStream
import kotlin.math.min

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AccountScreen(
    model: AppModel,
    onBack: () -> Unit,
    onFriends: () -> Unit,
    onChangeServer: () -> Unit,
) {
    val status by model.status.collectAsState()
    val user = status?.user
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    val snack = remember { SnackbarHostState() }
    var displayName by remember(user?.id) { mutableStateOf(user?.displayName.orEmpty()) }
    var uploading by remember { mutableStateOf(false) }

    val picker = rememberLauncherForActivityResult(ActivityResultContracts.PickVisualMedia()) { uri ->
        if (uri == null) return@rememberLauncherForActivityResult
        uploading = true
        scope.launch {
            runCatching {
                val bytes = withContext(Dispatchers.Default) { squareAvatar(context, uri) }
                model.api.uploadAvatar(bytes, "image/webp")
            }.onSuccess { model.updateUser(it) }
                .onFailure { snack.showSnackbar(it.message ?: "Could not upload the photo") }
            uploading = false
        }
    }

    if (user == null) {
        LaunchedEffect(Unit) { onBack() }
        return
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Account") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Back") } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background),
            )
        },
        snackbarHost = { SnackbarHost(snack) },
    ) { padding ->
        Column(
            Modifier.padding(padding).fillMaxSize().verticalScroll(rememberScrollState()).padding(horizontal = 16.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Spacer(Modifier.height(12.dp))
            Box {
                Avatar(user.name, model.api.avatarUrl(user.id, user.avatar), 96.dp)
                FilledIconButton(
                    onClick = { picker.launch(PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)) },
                    enabled = !uploading,
                    shape = CircleShape,
                    modifier = Modifier.align(Alignment.BottomEnd).size(34.dp),
                ) {
                    if (uploading) CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp, color = Color.White)
                    else Icon(Icons.Outlined.PhotoCamera, "Change photo", Modifier.size(18.dp))
                }
            }
            Spacer(Modifier.height(12.dp))
            Text(user.name, style = MaterialTheme.typography.titleLarge)
            Text("@${user.username} · ${user.role}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            if (user.avatar > 0) {
                TextButton(onClick = {
                    scope.launch {
                        runCatching { model.api.deleteAvatar() }
                            .onSuccess { model.updateUser(it) }
                            .onFailure { snack.showSnackbar(it.message ?: "Could not remove the photo") }
                    }
                }) { Text("Remove photo") }
            }

            Spacer(Modifier.height(20.dp))
            OutlinedTextField(
                value = displayName,
                onValueChange = { if (it.length <= 64) displayName = it },
                label = { Text("Display name") },
                supportingText = { Text("Shown to others and used as your name inside rooms") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Button(
                onClick = {
                    scope.launch {
                        runCatching { model.api.updateAccount(displayName.trim()) }
                            .onSuccess { model.updateUser(it); snack.showSnackbar("Saved") }
                            .onFailure { snack.showSnackbar(it.message ?: "Could not save") }
                    }
                },
                enabled = displayName.isNotBlank() && displayName.trim() != user.displayName,
                modifier = Modifier.align(Alignment.End),
            ) { Text("Save") }

            Spacer(Modifier.height(16.dp))
            HorizontalDivider(color = MaterialTheme.colorScheme.outlineVariant)
            if (status?.policy?.friendsEnabled == true) {
                ListItem(
                    headlineContent = { Text("Friends") },
                    trailingContent = { Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null) },
                    colors = ListItemDefaults.colors(containerColor = Color.Transparent),
                    modifier = Modifier.clickableRow(onFriends),
                )
            }
            ListItem(
                headlineContent = { Text("Server") },
                supportingContent = { Text(model.api.base?.toString().orEmpty()) },
                trailingContent = { Icon(Icons.AutoMirrored.Outlined.KeyboardArrowRight, null) },
                colors = ListItemDefaults.colors(containerColor = Color.Transparent),
                modifier = Modifier.clickableRow(onChangeServer),
            )
            Spacer(Modifier.height(20.dp))
            OutlinedButton(onClick = { model.signOut(); onBack() }, modifier = Modifier.fillMaxWidth()) { Text("Sign out") }
            Spacer(Modifier.height(32.dp))
        }
    }
}

private fun Modifier.clickableRow(onClick: () -> Unit): Modifier = this.clickable(onClick = onClick)

// squareAvatar crops the middle of the picked image to a 256px WebP
private fun squareAvatar(context: Context, uri: Uri): ByteArray {
    val source: Bitmap = if (Build.VERSION.SDK_INT >= 28) {
        ImageDecoder.decodeBitmap(ImageDecoder.createSource(context.contentResolver, uri)) { decoder, info, _ ->
            decoder.allocator = ImageDecoder.ALLOCATOR_SOFTWARE
            val longest = maxOf(info.size.width, info.size.height)
            if (longest > 1024) {
                val s = 1024f / longest
                decoder.setTargetSize((info.size.width * s).toInt(), (info.size.height * s).toInt())
            }
        }
    } else {
        context.contentResolver.openInputStream(uri).use { BitmapFactory.decodeStream(it) }
            ?: throw IllegalArgumentException("Could not read the image")
    }
    val side = min(source.width, source.height)
    val square = Bitmap.createBitmap(source, (source.width - side) / 2, (source.height - side) / 2, side, side)
    val scaled = Bitmap.createScaledBitmap(square, min(256, side), min(256, side), true)
    val out = ByteArrayOutputStream()
    @Suppress("DEPRECATION")
    val format = if (Build.VERSION.SDK_INT >= 30) Bitmap.CompressFormat.WEBP_LOSSY else Bitmap.CompressFormat.WEBP
    scaled.compress(format, 90, out)
    return out.toByteArray()
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FriendsScreen(model: AppModel, onBack: () -> Unit) {
    val scope = rememberCoroutineScope()
    val snack = remember { SnackbarHostState() }
    var friends by remember { mutableStateOf<List<Friend>?>(null) }
    var query by remember { mutableStateOf("") }
    var results by remember { mutableStateOf<List<PublicUser>>(emptyList()) }

    suspend fun load() {
        runCatching { model.api.friends() }.onSuccess { friends = it }
    }

    LaunchedEffect(Unit) {
        while (true) {
            load()
            delay(30_000)
        }
    }
    LaunchedEffect(query) {
        val q = query.trim()
        if (q.length < 2) {
            results = emptyList()
            return@LaunchedEffect
        }
        delay(250)
        runCatching { model.api.searchUsers(q) }.onSuccess { results = it }
    }

    fun act(block: suspend () -> Unit, done: String? = null) {
        scope.launch {
            runCatching { block() }
                .onSuccess { load(); if (done != null) snack.showSnackbar(done) }
                .onFailure { snack.showSnackbar(it.message ?: "Something went wrong") }
        }
    }

    val known = friends.orEmpty().map { it.user.id }.toSet()
    val incoming = friends.orEmpty().filter { it.status == "pending" && it.direction == "incoming" }
    val outgoing = friends.orEmpty().filter { it.status == "pending" && it.direction == "outgoing" }
    val accepted = friends.orEmpty().filter { it.status == "accepted" }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("Friends") },
                navigationIcon = { IconButton(onClick = onBack) { Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Back") } },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background),
            )
        },
        snackbarHost = { SnackbarHost(snack) },
    ) { padding ->
        LazyColumn(
            Modifier.padding(padding).fillMaxSize(),
            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
        ) {
            item {
                TextField(
                    value = query,
                    onValueChange = { query = it },
                    singleLine = true,
                    placeholder = { Text("Find people by username") },
                    leadingIcon = { Icon(Icons.Outlined.Search, null) },
                    shape = RoundedCornerShape(50),
                    colors = TextFieldDefaults.colors(
                        focusedIndicatorColor = Color.Transparent,
                        unfocusedIndicatorColor = Color.Transparent,
                        focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                        unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainer,
                    ),
                    modifier = Modifier.fillMaxWidth().padding(bottom = 8.dp),
                )
            }
            items(results.filter { it.id !in known }, key = { "r" + it.id }) { u ->
                PersonRow(model, u, null) {
                    IconButton(onClick = { act({ model.api.addFriend(u.username) }, "Friend request sent to ${u.name}"); query = "" }) {
                        Icon(Icons.Outlined.PersonAdd, "Add friend")
                    }
                }
            }
            if (incoming.isNotEmpty()) {
                item { SectionTitle("Requests", incoming.size, badge = true) }
                items(incoming, key = { "i" + it.user.id }) { f ->
                    PersonRow(model, f.user, "Wants to be friends") {
                        Row {
                            IconButton(onClick = { act({ model.api.acceptFriend(f.user.id) }) }) { Icon(Icons.Outlined.Check, "Accept", tint = Live) }
                            IconButton(onClick = { act({ model.api.removeFriend(f.user.id) }) }) { Icon(Icons.Outlined.Close, "Decline") }
                        }
                    }
                }
            }
            if (outgoing.isNotEmpty()) {
                item { SectionTitle("Sent", outgoing.size) }
                items(outgoing, key = { "o" + it.user.id }) { f ->
                    PersonRow(model, f.user, "Request sent") {
                        TextButton(onClick = { act({ model.api.removeFriend(f.user.id) }) }) { Text("Cancel") }
                    }
                }
            }
            item { SectionTitle("Friends", accepted.size) }
            if (friends == null) {
                item { Centered(Modifier.height(120.dp)) { CircularProgressIndicator(strokeWidth = 2.dp) } }
            } else if (accepted.isEmpty()) {
                item {
                    Text(
                        "No friends yet. Search for someone above to see when they are watching.",
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.padding(vertical = 12.dp),
                    )
                }
            }
            items(accepted, key = { "a" + it.user.id }) { f ->
                PersonRow(model, f.user, "@" + f.user.username) {
                    TextButton(onClick = { act({ model.api.removeFriend(f.user.id) }) }) {
                        Text("Remove", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }
    }
}

@Composable
private fun SectionTitle(text: String, count: Int, badge: Boolean = false) {
    Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 16.dp, bottom = 6.dp)) {
        Text(text, style = MaterialTheme.typography.titleSmall)
        Spacer(Modifier.width(8.dp))
        if (badge) Badge { Text("$count") } else Text("$count", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
    }
}

@Composable
private fun PersonRow(model: AppModel, user: PublicUser, subtitle: String?, actions: @Composable () -> Unit) {
    ListItem(
        leadingContent = { Avatar(user.name, model.api.avatarUrl(user.id, user.avatar), 40.dp) },
        headlineContent = { Text(user.name) },
        supportingContent = subtitle?.let { { Text(it) } },
        trailingContent = actions,
        colors = ListItemDefaults.colors(containerColor = Color.Transparent),
    )
}
