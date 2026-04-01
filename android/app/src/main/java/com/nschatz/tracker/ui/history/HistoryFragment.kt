package com.nschatz.tracker.ui.history

import android.app.DatePickerDialog
import android.graphics.Color
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.CircleRepository
import com.nschatz.tracker.data.repository.LocationRepository
import com.nschatz.tracker.databinding.FragmentHistoryBinding
import kotlinx.coroutines.launch
import org.osmdroid.config.Configuration
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.Polyline
import java.text.SimpleDateFormat
import java.util.Calendar
import java.util.Date
import java.util.Locale

class HistoryFragment : Fragment() {

    private var _binding: FragmentHistoryBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var locationRepo: LocationRepository
    private lateinit var circleRepo: CircleRepository

    private val members: MutableList<CircleMember> = mutableListOf()
    private var selectedDate: Calendar = Calendar.getInstance()
    private var currentPath: Polyline? = null

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentHistoryBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        val apiClient = ApiClient(session)
        val database = TrackerDatabase.getInstance(requireContext())
        locationRepo = LocationRepository(apiClient, session, database)
        circleRepo = CircleRepository(apiClient)

        Configuration.getInstance().userAgentValue = requireContext().packageName
        binding.mapView.setTileSource(TileSourceFactory.MAPNIK)
        binding.mapView.setMultiTouchControls(true)
        binding.mapView.controller.setZoom(13.0)

        updateDateButton()

        binding.btnDate.setOnClickListener {
            DatePickerDialog(
                requireContext(),
                { _, year, month, day ->
                    selectedDate.set(year, month, day)
                    updateDateButton()
                    loadHistory()
                },
                selectedDate.get(Calendar.YEAR),
                selectedDate.get(Calendar.MONTH),
                selectedDate.get(Calendar.DAY_OF_MONTH)
            ).show()
        }

        binding.spinnerMember.setOnItemSelectedListener(
            object : android.widget.AdapterView.OnItemSelectedListener {
                override fun onItemSelected(
                    parent: android.widget.AdapterView<*>?,
                    view: View?,
                    position: Int,
                    id: Long
                ) {
                    loadHistory()
                }

                override fun onNothingSelected(parent: android.widget.AdapterView<*>?) {}
            }
        )

        loadMembers()
    }

    private fun updateDateButton() {
        val fmt = SimpleDateFormat("MMM d, yyyy", Locale.getDefault())
        val today = Calendar.getInstance()
        val isToday = selectedDate.get(Calendar.YEAR) == today.get(Calendar.YEAR) &&
            selectedDate.get(Calendar.DAY_OF_YEAR) == today.get(Calendar.DAY_OF_YEAR)
        binding.btnDate.text = if (isToday) "Today" else fmt.format(selectedDate.time)
    }

    private fun loadMembers() {
        val circleId = session.activeCircleId ?: return
        viewLifecycleOwner.lifecycleScope.launch {
            circleRepo.getMembers(circleId).onSuccess { list ->
                members.clear()
                members.addAll(list)
                val names = list.map { it.displayName }
                val adapter = ArrayAdapter(
                    requireContext(),
                    android.R.layout.simple_spinner_item,
                    names
                ).apply {
                    setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
                }
                binding.spinnerMember.adapter = adapter
                if (list.isNotEmpty()) loadHistory()
            }
        }
    }

    private fun loadHistory() {
        val circleId = session.activeCircleId ?: return
        val selectedIndex = binding.spinnerMember.selectedItemPosition
        if (selectedIndex < 0 || selectedIndex >= members.size) return

        val member = members[selectedIndex]
        val isoFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
        val dayStart = selectedDate.clone() as Calendar
        dayStart.set(Calendar.HOUR_OF_DAY, 0)
        dayStart.set(Calendar.MINUTE, 0)
        dayStart.set(Calendar.SECOND, 0)
        dayStart.set(Calendar.MILLISECOND, 0)
        val dayEnd = dayStart.clone() as Calendar
        dayEnd.add(Calendar.DAY_OF_MONTH, 1)

        val from = isoFormat.format(dayStart.time)
        val to = isoFormat.format(dayEnd.time)

        viewLifecycleOwner.lifecycleScope.launch {
            locationRepo.getHistory(member.userId, from, to).onSuccess { locations ->
                // Clear old path
                currentPath?.let { binding.mapView.overlays.remove(it) }
                currentPath = null

                if (locations.size < 2) {
                    binding.mapView.invalidate()
                    return@onSuccess
                }

                val points = locations.map { GeoPoint(it.lat, it.lng) }
                val polyline = Polyline(binding.mapView).apply {
                    setPoints(points)
                    outlinePaint.color = Color.BLUE
                    outlinePaint.strokeWidth = 6f
                }
                binding.mapView.overlays.add(polyline)
                currentPath = polyline

                val center = points.first()
                binding.mapView.controller.setCenter(center)
                binding.mapView.invalidate()
            }
        }
    }

    override fun onResume() {
        super.onResume()
        binding.mapView.onResume()
    }

    override fun onPause() {
        super.onPause()
        binding.mapView.onPause()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
