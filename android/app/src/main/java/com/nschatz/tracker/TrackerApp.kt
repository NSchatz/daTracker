package com.nschatz.tracker

import android.app.Application
import android.app.NotificationChannel
import android.app.NotificationManager
import org.osmdroid.config.Configuration

class TrackerApp : Application() {

    companion object {
        const val CHANNEL_LOCATION = "location_service"
        const val CHANNEL_PLACE_ALERTS = "place_alerts"
    }

    override fun onCreate() {
        super.onCreate()
        Configuration.getInstance().userAgentValue = packageName
        createNotificationChannels()
    }

    private fun createNotificationChannels() {
        val manager = getSystemService(NotificationManager::class.java)
        val locationChannel = NotificationChannel(
            CHANNEL_LOCATION, "Location Service",
            NotificationManager.IMPORTANCE_LOW
        ).apply { description = "Shows when location tracking is active" }
        val alertChannel = NotificationChannel(
            CHANNEL_PLACE_ALERTS, "Place Alerts",
            NotificationManager.IMPORTANCE_HIGH
        ).apply { description = "Geofence enter/leave notifications" }
        manager.createNotificationChannels(listOf(locationChannel, alertChannel))
    }
}
