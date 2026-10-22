package app.rooms.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Link
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material.icons.outlined.Weekend
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import app.rooms.data.Room
import app.rooms.data.parseInviteLink
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private data class Section(val title: String, val rooms: List<Room>)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    model: AppModel,
    onRoom: (name: String, invite: String?) -> Unit,
    onAccount: () -> Unit,
    onSignIn: () -> Unit,
    onChangeServer: () -> Unit,
) {
    val status by model.status.collectAsState()
    val user = status?.user
    val scope = rememberCoroutineScope()

    var rooms by remember { mutableStateOf<List<Room>?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var refreshing by remember { mutableStateOf(false) }
    var query by remember { mutableStateOf("") }
    var category by remember { mutableStateOf<String?>(null) }
    var bust by remember { mutableLongStateOf(System.currentTimeMillis() / 30000) }
    var linkDialog by remember { mutableStateOf(false) }

    suspend fun load() {
        runCatching { model.api.directory() }
            .onSuccess { rooms = it; error = null }
            .onFailure { error = it.message ?: "Could not load rooms" }
        bust = System.currentTimeMillis() / 30000
    }

    LaunchedEffect(user?.id) {
        while (true) {
            load()
            delay(10_000)
        }
    }

    val photoOf = { name: String -> rooms?.firstNotNullOfOrNull { it.avatars[name] }?.let { model.api.apiUrl(it) } }

    val visible = rooms.orEmpty().filter { r ->
        // sleeping rooms and upcoming parties are as good as running ones
        (r.running || r.sleeping || r.partyAt != null || r.isOwner) &&
            (category == null || r.category == category) &&
            (query.isBlank() || listOf(r.displayTitle, r.name, r.description, r.category, r.owner?.name).plus(r.members)
                .any { it?.contains(query.trim(), ignoreCase = true) == true })
    }
    val sections = buildList {
        val used = mutableSetOf<String>()
        fun take(title: String, pred: (Room) -> Boolean) {
            val list = visible.filter { it.id !in used && pred(it) }
            used += list.map { it.id }
            if (list.isNotEmpty()) add(Section(title, list))
        }
        take("Upcoming watch parties") { it.partyAt != null && !it.ready }
        take("Featured") { it.featured && it.visibility == "public" }
        take("Friends") { it.isFriend || it.friendsInside.isNotEmpty() }
        take("Public") { it.visibility == "public" && !it.isOwner }
        take("Your rooms") { it.isOwner }
        take("Other") { true }
    }
    val categories = rooms.orEmpty().map { it.category }.filter { it.isNotBlank() }.distinct().sorted()
    val live = rooms.orEmpty().count { it.ready }
    val watching = rooms.orEmpty().sumOf { it.viewers }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Column {
                        Text("WatchParty", style = MaterialTheme.typography.titleLarge)
                        if (rooms != null) {
                            Text(
                                "$live live · $watching watching",
                                style = MaterialTheme.typography.labelMedium,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    }
                },
                actions = {
                    IconButton(onClick = { linkDialog = true }) { Icon(Icons.Outlined.Link, "Join with a link") }
                    if (user != null) {
                        IconButton(onClick = onAccount) {
                            Avatar(user.name, model.api.avatarUrl(user.id, user.avatar), 32.dp)
                        }
                    } else {
                        TextButton(onClick = onSignIn) { Text("Sign in") }
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(containerColor = MaterialTheme.colorScheme.background),
            )
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = refreshing,
            onRefresh = { scope.launch { refreshing = true; load(); refreshing = false } },
            modifier = Modifier.padding(padding).fillMaxSize(),
        ) {
            LazyColumn(
                contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 4.dp, bottom = 32.dp),
                verticalArrangement = Arrangement.spacedBy(14.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                item {
                    TextField(
                        value = query,
                        onValueChange = { query = it },
                        singleLine = true,
                        placeholder = { Text("Search rooms and people") },
                        leadingIcon = { Icon(Icons.Outlined.Search, null) },
                        shape = RoundedCornerShape(50),
                        colors = TextFieldDefaults.colors(
                            focusedIndicatorColor = Color.Transparent,
                            unfocusedIndicatorColor = Color.Transparent,
                            focusedContainerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
                            unfocusedContainerColor = MaterialTheme.colorScheme.surfaceContainer,
                        ),
                        modifier = Modifier.fillMaxWidth(),
                    )
                }
                if (categories.isNotEmpty()) {
                    item {
                        LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            item {
                                FilterChip(selected = category == null, onClick = { category = null }, label = { Text("All") }, shape = RoundedCornerShape(50))
                            }
                            items(categories) { c ->
                                FilterChip(
                                    selected = category == c,
                                    onClick = { category = if (category == c) null else c },
                                    label = { Text(c) },
                                    shape = RoundedCornerShape(50),
                                )
                            }
                        }
                    }
                }

                when {
                    rooms == null && error == null -> item {
                        Centered(Modifier.height(240.dp)) { CircularProgressIndicator(strokeWidth = 2.dp) }
                    }
                    rooms == null -> item { Message(error ?: "", "Try again", { scope.launch { load() } }) }
                    sections.isEmpty() -> item {
                        Message(
                            if (query.isNotBlank() || category != null) "No rooms match your search." else "No rooms are open right now.",
                            null, null,
                        )
                    }
                }

                sections.forEach { section ->
                    item(key = "h-" + section.title) {
                        Row(verticalAlignment = Alignment.CenterVertically, modifier = Modifier.padding(top = 6.dp)) {
                            Text(section.title, style = MaterialTheme.typography.titleSmall)
                            Spacer(Modifier.size(8.dp))
                            Text("${section.rooms.size}", style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                    }
                    items(section.rooms, key = { section.title + it.id }) { room ->
                        RoomCard(
                            room = room,
                            thumb = model.api.thumbnailUrl(room.name, bust),
                            photoOf = photoOf,
                            ownerPhoto = room.owner?.let { model.api.avatarUrl(it.id, it.avatar) },
                            onClick = { onRoom(room.name, null) },
                        )
                    }
                }

                item {
                    TextButton(onClick = onChangeServer, modifier = Modifier.fillMaxWidth()) {
                        Text(model.api.base?.host ?: "", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }
    }

    if (linkDialog) {
        JoinLinkDialog(
            onDismiss = { linkDialog = false },
            onLink = { input ->
                val link = parseInviteLink(input)
                when {
                    link == null -> "That is not a room link"
                    link.server != null && link.server.host != model.api.base?.host -> "That link is for another server (${link.server.host})"
                    else -> {
                        linkDialog = false
                        onRoom(link.room, link.invite)
                        null
                    }
                }
            },
        )
    }
}

@Composable
private fun Message(text: String, action: String?, onAction: (() -> Unit)?) {
    Column(
        Modifier.fillMaxWidth().padding(vertical = 56.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Icon(Icons.Outlined.Weekend, null, tint = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.size(36.dp))
        Spacer(Modifier.height(12.dp))
        Text(text, color = MaterialTheme.colorScheme.onSurfaceVariant)
        if (action != null && onAction != null) {
            Spacer(Modifier.height(12.dp))
            Button(onClick = onAction) { Text(action) }
        }
    }
}

@Composable
private fun JoinLinkDialog(onDismiss: () -> Unit, onLink: (String) -> String?) {
    var text by remember { mutableStateOf("") }
    var error by remember { mutableStateOf<String?>(null) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Join with a link") },
        text = {
            OutlinedTextField(
                value = text,
                onValueChange = { text = it; error = null },
                placeholder = { Text("Paste a room or invite link") },
                isError = error != null,
                supportingText = error?.let { { Text(it) } },
                singleLine = true,
            )
        },
        confirmButton = { Button(onClick = { error = onLink(text) }, enabled = text.isNotBlank()) { Text("Open") } },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}
