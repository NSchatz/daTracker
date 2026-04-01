package com.nschatz.tracker.websocket

import com.google.gson.Gson
import com.nschatz.tracker.data.model.MemberLocation
import com.nschatz.tracker.data.prefs.SessionManager
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import java.util.concurrent.TimeUnit

class LocationWebSocketClient(
    private val sessionManager: SessionManager,
    private val onLocationUpdate: (MemberLocation) -> Unit
) {

    private val gson = Gson()
    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    private val client: OkHttpClient = OkHttpClient.Builder()
        .pingInterval(30, TimeUnit.SECONDS)
        .build()

    private var webSocket: WebSocket? = null
    private var shouldReconnect = false
    private var currentCircleId: String? = null

    fun connect(circleId: String) {
        currentCircleId = circleId
        shouldReconnect = true
        openConnection(circleId)
    }

    private fun openConnection(circleId: String) {
        val token = sessionManager.token ?: return
        val serverUrl = sessionManager.serverUrl.trimEnd('/')

        val wsUrl = serverUrl
            .replace("https://", "wss://")
            .replace("http://", "ws://")
            .plus("/api/ws/circles/$circleId/locations")

        val request = Request.Builder()
            .url(wsUrl)
            .addHeader("Authorization", "Bearer $token")
            .build()

        webSocket = client.newWebSocket(request, object : WebSocketListener() {
            override fun onMessage(webSocket: WebSocket, text: String) {
                try {
                    val location = gson.fromJson(text, MemberLocation::class.java)
                    onLocationUpdate(location)
                } catch (e: Exception) {
                    // Ignore unparseable messages
                }
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                scheduleReconnect()
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                scheduleReconnect()
            }
        })
    }

    private fun scheduleReconnect() {
        if (!shouldReconnect) return
        val circleId = currentCircleId ?: return
        scope.launch {
            delay(5_000L)
            if (shouldReconnect) {
                openConnection(circleId)
            }
        }
    }

    fun disconnect() {
        shouldReconnect = false
        webSocket?.close(1000, "Client disconnecting")
        webSocket = null
    }
}
