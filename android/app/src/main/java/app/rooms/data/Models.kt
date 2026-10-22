package app.rooms.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class User(
    val id: Long = 0,
    val username: String = "",
    @SerialName("display_name") val displayName: String = "",
    val email: String = "",
    val role: String = "user",
    val avatar: Long = 0,
) {
    val name: String get() = displayName.ifBlank { username }
}

@Serializable
data class PublicUser(
    val id: Long = 0,
    val username: String = "",
    @SerialName("display_name") val displayName: String = "",
    val avatar: Long = 0,
) {
    val name: String get() = displayName.ifBlank { username }
}

@Serializable
data class Policy(
    @SerialName("homepage_enabled") val homepageEnabled: Boolean = true,
    @SerialName("guests_can_browse") val guestsCanBrowse: Boolean = true,
    @SerialName("rooms_require_login") val roomsRequireLogin: Boolean = false,
    @SerialName("registration_enabled") val registrationEnabled: Boolean = false,
    @SerialName("registration_approval") val registrationApproval: Boolean = false,
    @SerialName("friends_enabled") val friendsEnabled: Boolean = false,
    @SerialName("thumbnails_enabled") val thumbnailsEnabled: Boolean = true,
    @SerialName("password_min_length") val passwordMinLength: Int = 8,
)

@Serializable
data class AuthStatus(
    @SerialName("setup_required") val setupRequired: Boolean = false,
    val user: User? = null,
    val policy: Policy = Policy(),
)

@Serializable
data class Room(
    val id: String = "",
    val name: String = "",
    val title: String = "",
    val description: String = "",
    val category: String = "",
    val visibility: String = "private",
    val featured: Boolean = false,
    val owner: PublicUser? = null,
    @SerialName("is_owner") val isOwner: Boolean = false,
    @SerialName("is_friend") val isFriend: Boolean = false,
    @SerialName("can_manage") val canManage: Boolean = false,
    @SerialName("can_join") val canJoin: Boolean = false,
    val running: Boolean = false,
    val ready: Boolean = false,
    val paused: Boolean = false,
    val viewers: Int = 0,
    val members: List<String> = emptyList(),
    @SerialName("friends_inside") val friendsInside: List<String> = emptyList(),
    val avatars: Map<String, String> = emptyMap(),
    @SerialName("has_thumbnail") val hasThumbnail: Boolean = false,
    @SerialName("invite_code") val inviteCode: String? = null,
    @SerialName("now_playing") val nowPlaying: String = "",
    val sleeping: Boolean = false,
    @SerialName("scheduled_at") val scheduledAt: String? = null,
    val rsvps: Int = 0,
    val going: Boolean = false,
    val guests: List<String> = emptyList(),
) {
    val displayTitle: String get() = title.ifBlank { name }

    // start of the watch party, while it is upcoming or on
    val partyAt: java.time.Instant? get() = scheduledAt?.let { runCatching { java.time.Instant.parse(it) }.getOrNull() }

    // a party about to start, joining starts the room
    val partyOn: Boolean get() = partyAt?.let { it.minusSeconds(120).isBefore(java.time.Instant.now()) } == true

    val canWakeUp: Boolean get() = !running && (sleeping || partyOn)
}

@Serializable
data class Friend(
    val user: PublicUser = PublicUser(),
    val status: String = "pending",
    val direction: String = "incoming",
)

@Serializable
internal data class JoinRequest(
    val invite: String? = null,
    @SerialName("display_name") val displayName: String? = null,
)

@Serializable
internal data class JoinResponse(val url: String = "", val waking: Boolean = false)

@Serializable
internal data class LoginRequest(val username: String, val password: String)

@Serializable
internal data class RegisterRequest(
    val username: String,
    val password: String,
    @SerialName("display_name") val displayName: String? = null,
)

@Serializable
internal data class AccountUpdate(@SerialName("display_name") val displayName: String)

@Serializable
internal data class FriendRequest(val username: String)
