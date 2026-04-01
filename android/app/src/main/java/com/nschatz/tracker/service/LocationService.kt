package com.nschatz.tracker.service

import android.app.Notification
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.google.android.gms.location.ActivityRecognition
import com.google.android.gms.location.ActivityTransition
import com.google.android.gms.location.ActivityTransitionRequest
import com.google.android.gms.location.DetectedActivity
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.nschatz.tracker.R
import com.nschatz.tracker.TrackerApp
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.PendingLocationEntity
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.LocationPoint
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.LocationRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

class LocationService : Service() {

    companion object {
        const val INTERVAL_STATIONARY = 300_000L
        const val INTERVAL_WALKING = 30_000L
        const val INTERVAL_DRIVING = 10_000L

        const val ACTION_START = "com.nschatz.tracker.action.START"
        const val ACTION_STOP = "com.nschatz.tracker.action.STOP"
        const val ACTION_UPDATE_INTERVAL = "com.nschatz.tracker.action.UPDATE_INTERVAL"
        const val EXTRA_INTERVAL = "extra_interval"

        private const val NOTIFICATION_ID = 1001
    }

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())

    private lateinit var fusedLocationClient: FusedLocationProviderClient
    private lateinit var locationRepo: LocationRepository
    private lateinit var sessionManager: SessionManager

    private var currentInterval = INTERVAL_STATIONARY

    private val locationCallback = object : LocationCallback() {
        override fun onLocationResult(result: LocationResult) {
            val location = result.lastLocation ?: return
            val circleId = sessionManager.activeCircleId ?: return

            val isoFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US).apply {
                timeZone = TimeZone.getTimeZone("UTC")
            }
            val point = LocationPoint(
                lat = location.latitude,
                lng = location.longitude,
                speed = if (location.hasSpeed()) location.speed else null,
                batteryLevel = getBatteryLevel(),
                accuracy = if (location.hasAccuracy()) location.accuracy else null,
                recordedAt = isoFormat.format(Date(location.time))
            )

            serviceScope.launch {
                val result = locationRepo.submitLocations(circleId, listOf(point))
                if (result.isSuccess) {
                    locationRepo.flushQueue(circleId)
                } else {
                    locationRepo.queueLocation(
                        PendingLocationEntity(
                            lat = point.lat,
                            lng = point.lng,
                            speed = point.speed,
                            batteryLevel = point.batteryLevel,
                            accuracy = point.accuracy,
                            recordedAt = point.recordedAt
                        )
                    )
                }
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        fusedLocationClient = LocationServices.getFusedLocationProviderClient(this)
        sessionManager = SessionManager(this)
        val apiClient = ApiClient(sessionManager)
        val database = TrackerDatabase.getInstance(this)
        locationRepo = LocationRepository(apiClient, sessionManager, database)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START -> {
                startForeground(NOTIFICATION_ID, createNotification())
                startLocationUpdates()
                registerActivityTransitions()
            }
            ACTION_STOP -> {
                fusedLocationClient.removeLocationUpdates(locationCallback)
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
            ACTION_UPDATE_INTERVAL -> {
                val interval = intent.getLongExtra(EXTRA_INTERVAL, currentInterval)
                updateInterval(interval)
            }
        }
        return START_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun startLocationUpdates() {
        val request = LocationRequest.Builder(Priority.PRIORITY_HIGH_ACCURACY, currentInterval)
            .setMinUpdateIntervalMillis(currentInterval / 2)
            .build()

        try {
            fusedLocationClient.requestLocationUpdates(request, locationCallback, mainLooper)
        } catch (e: SecurityException) {
            stopSelf()
        }
    }

    fun updateInterval(ms: Long) {
        currentInterval = ms
        fusedLocationClient.removeLocationUpdates(locationCallback)
        startLocationUpdates()
    }

    private fun registerActivityTransitions() {
        val transitions = listOf(
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.STILL)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build(),
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.WALKING)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build(),
            ActivityTransition.Builder()
                .setActivityType(DetectedActivity.IN_VEHICLE)
                .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                .build()
        )

        val request = ActivityTransitionRequest(transitions)

        val intent = Intent(this, ActivityTransitionReceiver::class.java)
        val pendingIntent = PendingIntent.getBroadcast(
            this,
            0,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE
        )

        try {
            ActivityRecognition.getClient(this)
                .requestActivityTransitionUpdates(request, pendingIntent)
        } catch (e: SecurityException) {
            // Permission not granted; transitions won't be used
        }
    }

    private fun createNotification(): Notification {
        return NotificationCompat.Builder(this, TrackerApp.CHANNEL_LOCATION)
            .setContentTitle("Tracker Active")
            .setContentText("Sharing your location")
            .setSmallIcon(android.R.drawable.ic_menu_mylocation)
            .setOngoing(true)
            .setPriority(NotificationCompat.PRIORITY_LOW)
            .build()
    }

    private fun getBatteryLevel(): Int? {
        return try {
            val filter = IntentFilter(Intent.ACTION_BATTERY_CHANGED)
            val batteryStatus = registerReceiver(null, filter)
            val level = batteryStatus?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
            val scale = batteryStatus?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
            if (level >= 0 && scale > 0) (level * 100 / scale) else null
        } catch (e: Exception) {
            null
        }
    }
}
