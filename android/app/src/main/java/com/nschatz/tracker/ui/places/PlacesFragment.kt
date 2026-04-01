package com.nschatz.tracker.ui.places

import android.os.Bundle
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.fragment.app.Fragment
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.DividerItemDecoration
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.Geofence
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.GeofenceRepository
import com.nschatz.tracker.databinding.DialogGeofenceEditBinding
import com.nschatz.tracker.databinding.FragmentPlacesBinding
import com.nschatz.tracker.databinding.ItemGeofenceBinding
import kotlinx.coroutines.launch
import org.osmdroid.config.Configuration
import org.osmdroid.events.MapEventsReceiver
import org.osmdroid.tileprovider.tilesource.TileSourceFactory
import org.osmdroid.util.GeoPoint
import org.osmdroid.views.overlay.MapEventsOverlay
import org.osmdroid.views.overlay.Marker

class PlacesFragment : Fragment() {

    private var _binding: FragmentPlacesBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager
    private lateinit var geofenceRepo: GeofenceRepository

    private val geofences: MutableList<Geofence> = mutableListOf()
    private lateinit var adapter: GeofenceAdapter

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentPlacesBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())
        geofenceRepo = GeofenceRepository(ApiClient(session))

        adapter = GeofenceAdapter(geofences) { geofence ->
            deleteGeofence(geofence)
        }

        binding.recyclerGeofences.layoutManager = LinearLayoutManager(requireContext())
        binding.recyclerGeofences.addItemDecoration(
            DividerItemDecoration(requireContext(), DividerItemDecoration.VERTICAL)
        )
        binding.recyclerGeofences.adapter = adapter

        binding.fabAdd.setOnClickListener { showAddDialog() }

        loadGeofences()
    }

    private fun loadGeofences() {
        val circleId = session.activeCircleId ?: return
        viewLifecycleOwner.lifecycleScope.launch {
            geofenceRepo.getAll(circleId).onSuccess { list ->
                geofences.clear()
                geofences.addAll(list)
                adapter.notifyDataSetChanged()
            }
        }
    }

    private fun showAddDialog() {
        val dialogBinding = DialogGeofenceEditBinding.inflate(layoutInflater)
        var pickedPoint: GeoPoint? = null
        var pickedMarker: Marker? = null

        Configuration.getInstance().userAgentValue = requireContext().packageName
        dialogBinding.mapPicker.setTileSource(TileSourceFactory.MAPNIK)
        dialogBinding.mapPicker.setMultiTouchControls(true)
        dialogBinding.mapPicker.controller.setZoom(13.0)

        val eventsReceiver = object : MapEventsReceiver {
            override fun singleTapConfirmedHelper(p: GeoPoint): Boolean {
                pickedPoint = p
                pickedMarker?.let { dialogBinding.mapPicker.overlays.remove(it) }
                val marker = Marker(dialogBinding.mapPicker).apply {
                    position = p
                    setAnchor(Marker.ANCHOR_CENTER, Marker.ANCHOR_BOTTOM)
                }
                dialogBinding.mapPicker.overlays.add(marker)
                pickedMarker = marker
                dialogBinding.mapPicker.invalidate()
                return true
            }

            override fun longPressHelper(p: GeoPoint): Boolean = false
        }
        dialogBinding.mapPicker.overlays.add(MapEventsOverlay(eventsReceiver))

        val dialog = AlertDialog.Builder(requireContext())
            .setTitle("Add Place")
            .setView(dialogBinding.root)
            .setPositiveButton("Create", null)
            .setNegativeButton("Cancel", null)
            .create()

        dialog.setOnShowListener {
            dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener {
                val name = dialogBinding.editName.text?.toString()?.trim() ?: ""
                val radiusStr = dialogBinding.editRadius.text?.toString()?.trim() ?: ""
                val point = pickedPoint

                if (name.isEmpty()) {
                    Toast.makeText(requireContext(), "Enter a place name", Toast.LENGTH_SHORT).show()
                    return@setOnClickListener
                }
                if (point == null) {
                    Toast.makeText(requireContext(), "Tap the map to pick a location", Toast.LENGTH_SHORT).show()
                    return@setOnClickListener
                }
                val radius = radiusStr.toDoubleOrNull() ?: 100.0
                val circleId = session.activeCircleId ?: return@setOnClickListener

                viewLifecycleOwner.lifecycleScope.launch {
                    geofenceRepo.create(circleId, name, point.latitude, point.longitude, radius)
                        .onSuccess {
                            dialog.dismiss()
                            loadGeofences()
                        }
                        .onFailure {
                            Toast.makeText(requireContext(), "Failed to create place", Toast.LENGTH_SHORT).show()
                        }
                }
            }
        }

        dialog.show()
    }

    private fun deleteGeofence(geofence: Geofence) {
        viewLifecycleOwner.lifecycleScope.launch {
            geofenceRepo.delete(geofence.id).onSuccess {
                loadGeofences()
            }.onFailure {
                Toast.makeText(requireContext(), "Failed to delete", Toast.LENGTH_SHORT).show()
            }
        }
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }

    inner class GeofenceAdapter(
        private val items: List<Geofence>,
        private val onDelete: (Geofence) -> Unit
    ) : RecyclerView.Adapter<GeofenceAdapter.ViewHolder>() {

        inner class ViewHolder(val binding: ItemGeofenceBinding) :
            RecyclerView.ViewHolder(binding.root)

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
            val b = ItemGeofenceBinding.inflate(
                LayoutInflater.from(parent.context), parent, false
            )
            return ViewHolder(b)
        }

        override fun onBindViewHolder(holder: ViewHolder, position: Int) {
            val item = items[position]
            holder.binding.txtName.text = item.name
            holder.binding.txtRadius.text = "${item.radiusMeters.toInt()} m"
            holder.binding.btnDelete.setOnClickListener { onDelete(item) }
        }

        override fun getItemCount() = items.size
    }
}
