package app.rooms.ui

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import app.rooms.WatchPartyApp
import app.rooms.data.AuthStatus
import app.rooms.data.User
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import okhttp3.HttpUrl

// AppModel holds what every screen needs: the server and who is signed in.
class AppModel : ViewModel() {
    private val app = WatchPartyApp.instance
    val api get() = app.api
    val prefs get() = app.prefs

    private val _status = MutableStateFlow<AuthStatus?>(null)
    val status: StateFlow<AuthStatus?> = _status

    val user: User? get() = _status.value?.user

    init {
        if (prefs.server != null) refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            runCatching { api.status() }.onSuccess { _status.value = it }
        }
    }

    // useServer checks the address and remembers it when it is a neko-rooms server
    suspend fun useServer(url: HttpUrl): Result<AuthStatus> {
        val previous = api.base
        api.base = url
        return runCatching { api.status() }
            .onSuccess {
                if (previous != url) app.cookies.clear()
                prefs.server = url
                _status.value = it
            }
            .onFailure { api.base = previous }
    }

    suspend fun signIn(username: String, password: String): Result<Unit> = runCatching {
        api.login(username, password)
        _status.value = api.status()
    }

    suspend fun register(username: String, password: String, displayName: String): Result<Boolean> = runCatching {
        val active = api.register(username, password, displayName)
        _status.value = api.status()
        active
    }

    fun signOut() {
        viewModelScope.launch {
            runCatching { api.logout() }
            app.cookies.clear()
            runCatching { api.status() }.onSuccess { _status.value = it }
        }
    }

    fun updateUser(user: User) {
        _status.value = _status.value?.copy(user = user)
    }
}
