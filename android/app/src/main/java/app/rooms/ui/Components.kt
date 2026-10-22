package app.rooms.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.People
import androidx.compose.material.icons.outlined.PlayArrow
import androidx.compose.material.icons.outlined.Public
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import app.rooms.data.Room
import coil.compose.AsyncImage

private val avatarColors = listOf(
    0xFF0F766E, 0xFF0369A1, 0xFF6D28D9, 0xFFBE185D, 0xFFB45309,
    0xFFBE123C, 0xFF15803D, 0xFF4338CA, 0xFFA21CAF, 0xFF3F3F46,
).map { Color(it) }

@Composable
fun Avatar(name: String, photo: String?, size: Dp, modifier: Modifier = Modifier, ring: Boolean = false) {
    val color = avatarColors[(name.lowercase().hashCode() and 0x7fffffff) % avatarColors.size]
    val initials = name.trim().split(Regex("[\\s._-]+")).filter { it.isNotEmpty() }.let { parts ->
        when {
            parts.isEmpty() -> "?"
            parts.size == 1 -> parts[0].take(2).uppercase()
            else -> (parts[0].take(1) + parts[1].take(1)).uppercase()
        }
    }
    Box(
        modifier
            .size(size)
            .then(if (ring) Modifier.border(2.dp, MaterialTheme.colorScheme.background, CircleShape) else Modifier)
            .clip(CircleShape)
            .background(color),
        contentAlignment = Alignment.Center,
    ) {
        Text(initials, color = Color.White, fontSize = (size.value * 0.4f).sp, fontWeight = FontWeight.SemiBold)
        if (photo != null) {
            AsyncImage(photo, contentDescription = name, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        }
    }
}

// AvatarStack shows a few people overlapping
@Composable
fun AvatarStack(names: List<String>, photoOf: (String) -> String?, size: Dp = 24.dp, max: Int = 4) {
    Row(verticalAlignment = Alignment.CenterVertically) {
        names.take(max).forEachIndexed { i, n ->
            Avatar(n, photoOf(n), size, Modifier.offset(x = (-6 * i).dp), ring = true)
        }
        if (names.size > max) {
            Text(
                "+${names.size - max}",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.offset(x = (-6 * max + 6).dp),
            )
        }
    }
}

// relativeTime says when something happens, e.g. "in 2 hours"
fun relativeTime(at: java.time.Instant, now: java.time.Instant = java.time.Instant.now()): String {
    val mins = java.time.Duration.between(now, at).toMinutes()
    fun plural(n: Long, unit: String) = if (n == 1L) "in 1 $unit" else "in $n ${unit}s"
    return when {
        mins <= 0 -> "now"
        mins < 60 -> plural(mins, "minute")
        mins < 36 * 60 -> plural((mins + 30) / 60, "hour")
        else -> plural((mins + 720) / 1440, "day")
    }
}

fun partyDate(at: java.time.Instant): String =
    java.time.format.DateTimeFormatter.ofLocalizedDateTime(java.time.format.FormatStyle.MEDIUM, java.time.format.FormatStyle.SHORT)
        .withZone(java.time.ZoneId.systemDefault())
        .format(at)

@Composable
fun StatusPill(room: Room, modifier: Modifier = Modifier) {
    val party = room.partyAt
    val (label, color) = when {
        room.ready -> "Live" to Live
        room.paused -> "Paused" to Color(0xFFFBBF24)
        party != null -> "Party ${relativeTime(party)}" to MaterialTheme.colorScheme.primary
        room.sleeping -> "Sleeping" to Color(0xFF93C5FD)
        else -> "Offline" to Color(0xFFA1A1AA)
    }
    Row(
        modifier
            .clip(RoundedCornerShape(50))
            .background(Color.Black.copy(alpha = 0.55f))
            .padding(horizontal = 8.dp, vertical = 3.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(5.dp),
    ) {
        Box(Modifier.size(6.dp).clip(CircleShape).background(color))
        Text(label, color = Color.White, style = MaterialTheme.typography.labelSmall)
        if (room.ready) {
            Icon(Icons.Outlined.Visibility, null, tint = Color.White, modifier = Modifier.size(12.dp))
            Text("${room.viewers}", color = Color.White, style = MaterialTheme.typography.labelSmall)
        }
    }
}

fun visibilityIcon(room: Room) = when (room.visibility) {
    "public" -> Icons.Outlined.Public
    "friends" -> Icons.Outlined.People
    else -> Icons.Outlined.Lock
}

// RoomThumb shows the live screenshot, or a quiet placeholder
@Composable
fun RoomThumb(room: Room, thumb: String?, modifier: Modifier = Modifier) {
    Box(
        modifier
            .fillMaxWidth()
            .aspectRatio(16f / 9f)
            .clip(MaterialTheme.shapes.medium)
            .background(
                Brush.linearGradient(listOf(Color(0xFF1B1B1E), Color(0xFF0B0B0D)))
            ),
    ) {
        if (thumb != null && room.ready && room.hasThumbnail) {
            AsyncImage(thumb, contentDescription = null, contentScale = ContentScale.Crop, modifier = Modifier.fillMaxSize())
        }
        StatusPill(room, Modifier.align(Alignment.TopStart).padding(8.dp))
        Icon(
            visibilityIcon(room), null, tint = Color.White.copy(alpha = 0.85f),
            modifier = Modifier.align(Alignment.TopEnd).padding(10.dp).size(16.dp),
        )
    }
}

@Composable
fun RoomCard(room: Room, thumb: String?, photoOf: (String) -> String?, ownerPhoto: String?, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        shape = MaterialTheme.shapes.large,
        color = MaterialTheme.colorScheme.surfaceContainerLow,
        modifier = Modifier
            .fillMaxWidth()
            .border(1.dp, MaterialTheme.colorScheme.outlineVariant, MaterialTheme.shapes.large),
    ) {
        Column(Modifier.padding(6.dp)) {
            RoomThumb(room, thumb)
            Column(Modifier.padding(start = 8.dp, end = 8.dp, top = 10.dp, bottom = 8.dp)) {
                Text(room.displayTitle, style = MaterialTheme.typography.titleMedium, maxLines = 1, overflow = TextOverflow.Ellipsis)
                if (room.nowPlaying.isNotBlank()) {
                    Spacer(Modifier.height(4.dp))
                    NowPlaying(room.nowPlaying)
                }
                Spacer(Modifier.height(6.dp))
                Row(verticalAlignment = Alignment.CenterVertically) {
                    val owner = room.owner
                    if (owner != null) {
                        Avatar(owner.name, ownerPhoto, 18.dp)
                        Spacer(Modifier.width(6.dp))
                        Text(owner.name, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, maxLines = 1)
                    }
                    if (room.category.isNotBlank()) {
                        Text(
                            (if (owner != null) "  ·  " else "") + room.category,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1,
                        )
                    }
                    Spacer(Modifier.weight(1f))
                    if (room.members.isNotEmpty()) AvatarStack(room.members, photoOf, 22.dp, 3)
                }
            }
        }
    }
}

@Composable
fun NowPlaying(title: String, modifier: Modifier = Modifier) {
    Row(modifier, verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Outlined.PlayArrow, null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(14.dp))
        Spacer(Modifier.width(4.dp))
        Text(title, style = MaterialTheme.typography.bodySmall, maxLines = 1, overflow = TextOverflow.Ellipsis)
    }
}

@Composable
fun NicknameDialog(initial: String, onDismiss: () -> Unit, onConfirm: (String) -> Unit) {
    var name by remember { mutableStateOf(initial) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("Pick a nickname") },
        text = {
            OutlinedTextField(
                value = name,
                onValueChange = { if (it.length <= 32) name = it },
                singleLine = true,
                placeholder = { Text("Your name") },
            )
        },
        confirmButton = {
            Button(onClick = { onConfirm(name.trim()) }, enabled = name.isNotBlank()) { Text("Join") }
        },
        dismissButton = { TextButton(onClick = onDismiss) { Text("Cancel") } },
    )
}

@Composable
fun Centered(modifier: Modifier = Modifier, content: @Composable () -> Unit) {
    Box(modifier.fillMaxSize(), contentAlignment = Alignment.Center) { content() }
}
