package com.nschatz.tracker.service

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.google.android.gms.location.ActivityTransition
import com.google.android.gms.location.ActivityTransitionResult
import com.google.android.gms.location.DetectedActivity

class ActivityTransitionReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (!ActivityTransitionResult.hasResult(intent)) return

        val result = ActivityTransitionResult.extractResult(intent) ?: return

        for (event in result.transitionEvents) {
            if (event.transitionType != ActivityTransition.ACTIVITY_TRANSITION_ENTER) continue

            val interval = when (event.activityType) {
                DetectedActivity.STILL -> LocationService.INTERVAL_STATIONARY
                DetectedActivity.WALKING,
                DetectedActivity.ON_FOOT,
                DetectedActivity.RUNNING -> LocationService.INTERVAL_WALKING
                DetectedActivity.IN_VEHICLE,
                DetectedActivity.ON_BICYCLE -> LocationService.INTERVAL_DRIVING
                else -> continue
            }

            val serviceIntent = Intent(context, LocationService::class.java).apply {
                action = LocationService.ACTION_UPDATE_INTERVAL
                putExtra(LocationService.EXTRA_INTERVAL, interval)
            }
            context.startService(serviceIntent)
            break
        }
    }
}
