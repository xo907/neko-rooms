package app.rooms.ui

import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.systemBarsPadding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material.icons.outlined.Visibility
import androidx.compose.material.icons.outlined.VisibilityOff
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import app.rooms.R
import kotlinx.coroutines.launch

@Composable
private fun AuthFrame(title: String, onBack: () -> Unit, content: @Composable () -> Unit) {
    Column(Modifier.fillMaxSize().systemBarsPadding().imePadding()) {
        IconButton(onClick = onBack, modifier = Modifier.padding(4.dp)) {
            Icon(Icons.AutoMirrored.Outlined.ArrowBack, "Back")
        }
        Column(
            Modifier
                .fillMaxSize()
                .verticalScroll(rememberScrollState())
                .padding(horizontal = 24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(12.dp),
        ) {
            Spacer(Modifier.height(16.dp))
            Image(painterResource(R.drawable.logo), null, Modifier.size(56.dp))
            Text(title, style = MaterialTheme.typography.headlineMedium)
            Spacer(Modifier.height(8.dp))
            content()
        }
    }
}

@Composable
private fun PasswordField(value: String, onChange: (String) -> Unit, label: String, action: ImeAction, onAction: () -> Unit) {
    var show by remember { mutableStateOf(false) }
    OutlinedTextField(
        value = value,
        onValueChange = onChange,
        label = { Text(label) },
        singleLine = true,
        visualTransformation = if (show) VisualTransformation.None else PasswordVisualTransformation(),
        trailingIcon = {
            IconButton(onClick = { show = !show }) {
                Icon(if (show) Icons.Outlined.VisibilityOff else Icons.Outlined.Visibility, if (show) "Hide" else "Show")
            }
        },
        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password, imeAction = action),
        keyboardActions = KeyboardActions(onDone = { onAction() }, onNext = { onAction() }),
        modifier = Modifier.fillMaxWidth(),
    )
}

@Composable
fun LoginScreen(model: AppModel, onDone: () -> Unit, onRegister: () -> Unit, onBack: () -> Unit) {
    val status by model.status.collectAsState()
    val scope = rememberCoroutineScope()
    var username by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    fun submit() {
        if (username.isBlank() || password.isEmpty() || busy) return
        busy = true
        error = null
        scope.launch {
            model.signIn(username.trim(), password)
                .onSuccess { onDone() }
                .onFailure { error = it.message ?: "Could not sign in" }
            busy = false
        }
    }

    AuthFrame("Sign in", onBack) {
        OutlinedTextField(
            value = username,
            onValueChange = { username = it },
            label = { Text("Username") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(autoCorrectEnabled = false, imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        PasswordField(password, { password = it }, "Password", ImeAction.Done) { submit() }
        if (error != null) Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium)
        Button(onClick = { submit() }, enabled = !busy, modifier = Modifier.fillMaxWidth().height(50.dp)) {
            if (busy) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp) else Text("Sign in")
        }
        if (status?.policy?.registrationEnabled == true) {
            TextButton(onClick = onRegister) { Text("New here? Create an account") }
        }
    }
}

@Composable
fun RegisterScreen(model: AppModel, onDone: () -> Unit, onBack: () -> Unit) {
    val status by model.status.collectAsState()
    val minLength = status?.policy?.passwordMinLength ?: 8
    val scope = rememberCoroutineScope()
    var username by remember { mutableStateOf("") }
    var displayName by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var password2 by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    var pending by remember { mutableStateOf(false) }

    fun submit() {
        if (busy) return
        when {
            username.isBlank() -> { error = "Pick a username"; return }
            password.length < minLength -> { error = "Use at least $minLength characters"; return }
            password != password2 -> { error = "Passwords do not match"; return }
        }
        busy = true
        error = null
        scope.launch {
            model.register(username.trim(), password, displayName.trim())
                .onSuccess { active -> if (active) onDone() else pending = true }
                .onFailure { error = it.message ?: "Could not create the account" }
            busy = false
        }
    }

    AuthFrame("Create account", onBack) {
        if (pending) {
            Text(
                "Your account was created and is waiting for approval by an administrator.",
                textAlign = TextAlign.Center,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Button(onClick = onBack, modifier = Modifier.fillMaxWidth()) { Text("OK") }
            return@AuthFrame
        }
        OutlinedTextField(
            value = username, onValueChange = { username = it }, label = { Text("Username") }, singleLine = true,
            keyboardOptions = KeyboardOptions(autoCorrectEnabled = false, imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        OutlinedTextField(
            value = displayName, onValueChange = { displayName = it }, label = { Text("Display name (optional)") }, singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Next),
            modifier = Modifier.fillMaxWidth(),
        )
        PasswordField(password, { password = it }, "Password", ImeAction.Next) {}
        PasswordField(password2, { password2 = it }, "Repeat password", ImeAction.Done) { submit() }
        if (error != null) Text(error!!, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium)
        Button(onClick = { submit() }, enabled = !busy, modifier = Modifier.fillMaxWidth().height(50.dp)) {
            if (busy) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp) else Text("Create account")
        }
    }
}
