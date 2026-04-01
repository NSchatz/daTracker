package com.nschatz.tracker.fcm

import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import androidx.core.app.NotificationCompat
import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.nschatz.tracker.TrackerApp
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.FcmTokenRequest
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

class TrackerFirebaseService : FirebaseMessagingService() {

    private val scope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    override fun onNewToken(token: String) {
        super.onNewToken(token)
        val sessionManager = SessionManager(this)
        if (sessionManager.isLoggedIn) {
            val apiClient = ApiClient(sessionManager)
            scope.launch {
                try {
                    apiClient.fcm.registerToken(FcmTokenRequest(token))
                } catch (e: Exception) {
                    // Silently fail; token will be re-registered on next login
                }
            }
        }
    }

    override fun onMessageReceived(message: RemoteMessage) {
        super.onMessageReceived(message)

        val notification = message.notification ?: return
        val title = notification.title ?: "Tracker"
        val body = notification.body ?: ""

        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        }
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )

        val notificationId = System.currentTimeMillis().toInt()
        val builtNotification = NotificationCompat.Builder(this, TrackerApp.CHANNEL_PLACE_ALERTS)
            .setContentTitle(title)
            .setContentText(body)
            .setSmallIcon(android.R.drawable.ic_dialog_info)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setAutoCancel(true)
            .setContentIntent(pendingIntent)
            .build()

        val manager = getSystemService(NotificationManager::class.java)
        manager.notify(notificationId, builtNotification)
    }
}
