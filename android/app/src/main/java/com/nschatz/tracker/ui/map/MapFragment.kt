package com.nschatz.tracker.ui.map

import android.graphics.Color
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.MemberLocation
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.data.repository.GeofenceRepository
import com.nschatz.tracker.data.repository.LocationRepository
import com.nschatz.tracker.databinding.FragmentMapBinding
import com.nschatz.tracker.websocket.LocationWebSocketClient
import kotlinx.coroutines.launch
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.Marker
import org.osmdroid.views.overlay.Polygon

class MapFragment : Fragment() {

    private var _binding: FragmentMapBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var apiClient: ApiClient
    private lateinit var locationRepo: LocationRepository
    private lateinit var circleRepo: CircleRepository
    private lateinit var geofenceRepo: GeofenceRepository

    private var webSocketClient: LocationWebSocketClient? = null
    private val memberMarkers: MutableMap<String, Marker> = mutableMapOf()
    private val geofenceOverlays: MutableList<Polygon> = mutableListOf()

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentMapBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        apiClient = ApiClient(session)
        val database = TrackerDatabase.getInstance(requireContext())
        locationRepo = LocationRepository(apiClient, session, database)
        circleRepo = CircleRepository(apiClient)
        geofenceRepo = GeofenceRepository(apiClient)

        Configuration.getInstance().userAgentValue = requireContext().packageName

        binding.mapView.setTileSource(TileSourceFactory.MAPNIK)
        binding.mapView.setMultiTouchControls(true)
        binding.mapView.controller.setZoom(15.0)

        loadData()
        connectWebSocket()
    }

    private fun loadData() {
        val circleId = session.activeCircleId ?: return
        binding.progress.visibility = View.VISIBLE

        viewLifecycleOwner.lifecycleScope.launch {
            val displayNames = mutableMapOf<String, String>()

            circleRepo.getMembers(circleId).onSuccess { members ->
                members.forEach { displayNames[it.userId] = it.displayName }
            }

            locationRepo.getLatest(circleId).onSuccess { locations ->
                locations.forEach { updateMarker(it) }
                if (locations.isNotEmpty()) {
                    val first = locations.first()
                    binding.mapView.controller.setCenter(GeoPoint(first.lat, first.lng))
                }
            }

            geofenceRepo.getAll(circleId).onSuccess { geofences ->
                drawGeofences(geofences)
            }

            binding.progress.visibility = View.GONE
        }
    }

    private fun connectWebSocket() {
        val circleId = session.activeCircleId ?: return
        webSocketClient = LocationWebSocketClient(session) { location ->
            activity?.runOnUiThread { updateMarker(location) }
        }
        webSocketClient?.connect(circleId)
    }

    private fun updateMarker(location: MemberLocation) {
        val map = binding.mapView
        val position = GeoPoint(location.lat, location.lng)
        val timeStr = location.recordedAt.take(16).replace("T", " ")
        val batteryStr = location.batteryLevel?.let { " · $it%" } ?: ""

        val existing = memberMarkers[location.userId]
        if (existing != null) {
            existing.position = position
            existing.snippet = "$timeStr$batteryStr"
        } else {
            val marker = Marker(map)
            marker.position = position
            marker.title = location.displayName
            marker.snippet = "$timeStr$batteryStr"
            marker.setAnchor(Marker.ANCHOR_CENTER, Marker.ANCHOR_BOTTOM)
            map.overlays.add(marker)
            memberMarkers[location.userId] = marker
        }
        map.invalidate()
    }

    private fun drawGeofences(geofences: List<com.nschatz.tracker.data.model.Geofence>) {
        val map = binding.mapView
        geofenceOverlays.forEach { map.overlays.remove(it) }
        geofenceOverlays.clear()

        geofences.forEach { geofence ->
            val center = GeoPoint(geofence.lat, geofence.lng)
            val points = Polygon.pointsAsCircle(center, geofence.radiusMeters)
            val polygon = Polygon(map).apply {
                this.points = points
                fillPaint.color = Color.parseColor("#331B73E8")
                outlinePaint.color = Color.parseColor("#881B73E8")
                outlinePaint.strokeWidth = 3f
                title = geofence.name
            }
            map.overlays.add(polygon)
            geofenceOverlays.add(polygon)
        }
        map.invalidate()
    }

    override fun onResume() {
        super.onResume()
        binding.mapView.onResume()
        connectWebSocket()
    }

    override fun onPause() {
        super.onPause()
        binding.mapView.onPause()
        webSocketClient?.disconnect()
        webSocketClient = null
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
