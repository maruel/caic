// ViewModel for the Settings screen, managing connection testing and preference updates.
package com.fghbuild.caic.ui.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.caic.sdk.v1.ApiClient
import com.fghbuild.caic.data.SettingsRepository
import com.fghbuild.caic.data.SettingsState
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.FlowPreview
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.flow.drop
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

enum class ConnectionStatus { Idle, Testing, Success, Failed }

data class SettingsScreenState(
    val settings: SettingsState = SettingsState(),
    val connectionStatus: ConnectionStatus = ConnectionStatus.Idle,
)

private const val DEBOUNCE_MS = 500L

@OptIn(FlowPreview::class)
@HiltViewModel
class SettingsViewModel @Inject constructor(
    private val settingsRepository: SettingsRepository,
    @Suppress("UnusedPrivateProperty") private val apiClient: ApiClient,
) : ViewModel() {
    private val _state = MutableStateFlow(SettingsScreenState())
    val state: StateFlow<SettingsScreenState> = _state.asStateFlow()

    // Local buffer for the URL text field so keystrokes aren't blocked by DataStore round-trips.
    private val serverURLDraft = MutableStateFlow("")

    init {
        viewModelScope.launch {
            settingsRepository.settings.collect { settings ->
                _state.update { prev ->
                    // Seed the draft from DataStore on first load only.
                    if (prev.settings.serverURL.isEmpty() && settings.serverURL.isNotEmpty()) {
                        serverURLDraft.value = settings.serverURL
                    }
                    prev.copy(settings = settings.copy(serverURL = serverURLDraft.value))
                }
            }
        }
        // Debounce URL writes to DataStore.
        viewModelScope.launch {
            serverURLDraft.drop(1).debounce(DEBOUNCE_MS).collect { url ->
                settingsRepository.updateServerURL(url)
            }
        }
    }

    fun updateServerURL(url: String) {
        serverURLDraft.value = url
        _state.update { it.copy(settings = it.settings.copy(serverURL = url)) }
    }

    fun updateVoiceEnabled(enabled: Boolean) {
        viewModelScope.launch { settingsRepository.updateVoiceEnabled(enabled) }
    }

    fun updateVoiceName(name: String) {
        viewModelScope.launch { settingsRepository.updateVoiceName(name) }
    }

    fun testConnection() {
        val url = _state.value.settings.serverURL
        if (url.isBlank()) {
            _state.update { it.copy(connectionStatus = ConnectionStatus.Failed) }
            return
        }
        _state.update { it.copy(connectionStatus = ConnectionStatus.Testing) }
        viewModelScope.launch {
            try {
                val client = ApiClient(url)
                client.listHarnesses()
                _state.update { it.copy(connectionStatus = ConnectionStatus.Success) }
            } catch (_: Exception) {
                _state.update { it.copy(connectionStatus = ConnectionStatus.Failed) }
            }
        }
    }
}
