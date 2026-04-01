package com.nschatz.tracker.ui.main

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import android.util.Log
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.google.firebase.messaging.FirebaseMessaging
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.R
import com.nschatz.tracker.data.model.FcmTokenRequest
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.databinding.ActivityMainBinding
import com.nschatz.tracker.service.LocationService
import com.nschatz.tracker.ui.circle.CircleFragment
import com.nschatz.tracker.ui.history.HistoryFragment
import com.nschatz.tracker.ui.map.MapFragment
import com.nschatz.tracker.ui.places.PlacesFragment
import com.nschatz.tracker.ui.profile.ProfileFragment
import kotlinx.coroutines.launch

class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding
    private lateinit var session: SessionManager

    private val requestMultiplePermissions =
        registerForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { results ->
            val fineLocationGranted = results[Manifest.permission.ACCESS_FINE_LOCATION] == true
            if (fineLocationGranted) {
                requestBackgroundLocation()
            }
        }

    private val requestBackgroundLocationPermission =
        registerForActivityResult(ActivityResultContracts.RequestPermission()) { granted ->
            if (granted) {
                startLocationService()
            } else {
                // Start service anyway with foreground-only location
                startLocationService()
            }
        }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        session = SessionManager(this)

        initActiveCircle()
        setupBottomNav()
        requestPermissions()
        registerFcmToken()

        if (savedInstanceState == null) {
            binding.bottomNav.selectedItemId = R.id.nav_map
        }
    }

    private fun setupBottomNav() {
        binding.bottomNav.setOnItemSelectedListener { item ->
            val fragment: Fragment = when (item.itemId) {
                R.id.nav_map -> MapFragment()
                R.id.nav_history -> HistoryFragment()
                R.id.nav_places -> PlacesFragment()
                R.id.nav_circle -> CircleFragment()
                R.id.nav_profile -> ProfileFragment()
                else -> return@setOnItemSelectedListener false
            }
            supportFragmentManager.beginTransaction()
                .replace(R.id.fragmentContainer, fragment)
                .commit()
            true
        }
    }

    private fun requestPermissions() {
        val permissions = mutableListOf(
            Manifest.permission.ACCESS_FINE_LOCATION
        )
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            permissions.add(Manifest.permission.POST_NOTIFICATIONS)
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            permissions.add(Manifest.permission.ACTIVITY_RECOGNITION)
        }
        requestMultiplePermissions.launch(permissions.toTypedArray())
    }

    private fun requestBackgroundLocation() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            val bgGranted = ContextCompat.checkSelfPermission(
                this, Manifest.permission.ACCESS_BACKGROUND_LOCATION
            ) == PackageManager.PERMISSION_GRANTED

            if (!bgGranted) {
                requestBackgroundLocationPermission.launch(
                    Manifest.permission.ACCESS_BACKGROUND_LOCATION
                )
                return
            }
        }
        startLocationService()
    }

    private fun startLocationService() {
        val intent = Intent(this, LocationService::class.java).apply {
            action = LocationService.ACTION_START
        }
        ContextCompat.startForegroundService(this, intent)
    }

    private fun registerFcmToken() {
        FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
            lifecycleScope.launch {
                try {
                    val api = ApiClient(session)
                    api.fcm.registerToken(FcmTokenRequest(token))
                } catch (e: Exception) {
                    Log.w("MainActivity", "FCM token registration failed", e)
                }
            }
        }
    }

    private fun initActiveCircle() {
        if (session.activeCircleId != null) return
        lifecycleScope.launch {
            try {
                val api = ApiClient(session)
                val circleRepo = CircleRepository(api)
                circleRepo.getAll().onSuccess { circles ->
                    if (circles.isNotEmpty()) {
                        session.activeCircleId = circles.first().id
                    }
                }
            } catch (e: Exception) { /* retry next launch */ }
        }
    }
}
