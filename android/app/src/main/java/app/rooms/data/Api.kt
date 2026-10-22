package app.rooms.data

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.KSerializer
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import okhttp3.HttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.toRequestBody

class ApiException(val code: Int, message: String) : Exception(message)

val json = Json {
    ignoreUnknownKeys = true
    coerceInputValues = true
    explicitNulls = false
    encodeDefaults = false
}

private val jsonType = "application/json".toMediaType()

// Api talks to the neko-rooms server, the same endpoints the site uses.
class Api(private val http: OkHttpClient) {
    @Volatile
    var base: HttpUrl? = null

    private fun url(path: String): HttpUrl =
        (base ?: throw ApiException(0, "No server set")).resolve("api/$path")
            ?: throw ApiException(0, "Invalid server address")

    fun apiUrl(path: String): String = url(path).toString()

    fun thumbnailUrl(room: String, bust: Long, invite: String? = null): String =
        url("public/rooms/${enc(room)}/thumbnail.jpg").newBuilder()
            .addQueryParameter("t", bust.toString())
            .apply { if (invite != null) addQueryParameter("invite", invite) }
            .build().toString()

    fun avatarUrl(id: Long, version: Long): String? =
        if (version > 0) url("public/avatars/$id").newBuilder().addQueryParameter("v", version.toString()).build().toString() else null

    // links that open the room on the site, for sharing
    fun roomLink(room: String, invite: String? = null): String {
        val b = base ?: return ""
        return b.toString() + "r/" + enc(room) + (if (invite != null) "?invite=" + enc(invite) else "")
    }

    private suspend fun <T> call(request: Request, parse: (String) -> T): T = withContext(Dispatchers.IO) {
        http.newCall(request).execute().use { res ->
            val body = res.body?.string().orEmpty()
            if (!res.isSuccessful) {
                throw ApiException(res.code, body.trim().take(200).ifEmpty { "Server error (${res.code})" })
            }
            parse(body)
        }
    }

    private suspend fun <T> get(path: String, ser: KSerializer<T>): T =
        call(Request.Builder().url(url(path)).build()) { json.decodeFromString(ser, it) }

    private suspend fun <T> send(method: String, path: String, body: RequestBody?, ser: KSerializer<T>?): T? =
        call(Request.Builder().url(url(path)).method(method, body).build()) { text ->
            if (ser == null || text.isBlank()) null else json.decodeFromString(ser, text)
        }

    // exec sends a request whose response body is not needed
    private suspend fun exec(method: String, path: String, body: RequestBody?) {
        call(Request.Builder().url(url(path)).method(method, body).build()) { }
    }

    private fun <T> body(ser: KSerializer<T>, value: T): RequestBody =
        json.encodeToString(ser, value).toRequestBody(jsonType)

    private val empty: RequestBody get() = "{}".toRequestBody(jsonType)

    // auth

    suspend fun status(): AuthStatus = get("auth/status", AuthStatus.serializer())

    suspend fun login(username: String, password: String) {
        exec("POST", "auth/login", body(LoginRequest.serializer(), LoginRequest(username, password)))
    }

    suspend fun logout() {
        exec("POST", "auth/logout", empty)
    }

    // register returns false when the account waits for approval
    suspend fun register(username: String, password: String, displayName: String?): Boolean {
        val text = call(
            Request.Builder().url(url("auth/register"))
                .post(body(RegisterRequest.serializer(), RegisterRequest(username, password, displayName?.ifBlank { null })))
                .build()
        ) { it }
        return !json.parseToJsonElement(text).jsonObject.containsKey("pending_approval")
    }

    // account

    suspend fun updateAccount(displayName: String): User =
        send("PUT", "account", body(AccountUpdate.serializer(), AccountUpdate(displayName)), User.serializer())!!

    suspend fun uploadAvatar(image: ByteArray, mime: String): User {
        val form = MultipartBody.Builder().setType(MultipartBody.FORM)
            .addFormDataPart("file", "avatar", image.toRequestBody(mime.toMediaType()))
            .build()
        return send("POST", "account/avatar", form, User.serializer())!!
    }

    suspend fun deleteAvatar(): User = send("DELETE", "account/avatar", null, User.serializer())!!

    // rooms

    suspend fun directory(): List<Room> = get("public/rooms", ListSerializer(Room.serializer()))

    suspend fun room(name: String, invite: String? = null): Room {
        val u = url("public/rooms/${enc(name)}").newBuilder()
            .apply { if (invite != null) addQueryParameter("invite", invite) }.build()
        return call(Request.Builder().url(u).build()) { json.decodeFromString(Room.serializer(), it) }
    }

    // join returns the room address with credentials
    suspend fun join(name: String, invite: String?, displayName: String?): HttpUrl {
        val res = send(
            "POST", "public/rooms/${enc(name)}/join",
            body(JoinRequest.serializer(), JoinRequest(invite, displayName)), JoinResponse.serializer()
        )!!
        return base!!.resolve(res.url) ?: throw ApiException(0, "Invalid room address")
    }

    // friends

    suspend fun friends(): List<Friend> = get("friends", ListSerializer(Friend.serializer()))

    suspend fun addFriend(username: String) {
        exec("POST", "friends", body(FriendRequest.serializer(), FriendRequest(username)))
    }

    suspend fun acceptFriend(id: Long) {
        exec("POST", "friends/$id/accept", empty)
    }

    suspend fun removeFriend(id: Long) {
        exec("DELETE", "friends/$id", null)
    }

    suspend fun searchUsers(query: String): List<PublicUser> {
        val u = url("users/search").newBuilder().addQueryParameter("q", query).build()
        return call(Request.Builder().url(u).build()) { json.decodeFromString(ListSerializer(PublicUser.serializer()), it) }
    }
}

private fun enc(s: String): String = java.net.URLEncoder.encode(s, "UTF-8").replace("+", "%20")
