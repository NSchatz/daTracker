package com.nschatz.tracker.ui.profile

import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Toast
import androidx.fragment.app.Fragment
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.databinding.FragmentProfileBinding
import com.nschatz.tracker.service.LocationService
import com.nschatz.tracker.ui.auth.LoginActivity

class ProfileFragment : Fragment() {

    private var _binding: FragmentProfileBinding? = null
    private val binding get() = _binding!!

    private lateinit var session: SessionManager

    override fun onCreateView(
        inflater: LayoutInflater,
        container: ViewGroup?,
        savedInstanceState: Bundle?
    ): View {
        _binding = FragmentProfileBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        session = SessionManager(requireContext())

        binding.txtDisplayName.text = session.displayName ?: ""
        binding.txtEmail.text = session.email ?: ""
        binding.editServerUrl.setText(session.serverUrl)

        binding.btnSave.setOnClickListener {
            val url = binding.editServerUrl.text?.toString()?.trim() ?: ""
            if (url.isNotEmpty()) {
                session.serverUrl = url
            }
            Toast.makeText(requireContext(), "Saved", Toast.LENGTH_SHORT).show()
        }

        binding.btnLogout.setOnClickListener {
            stopLocationService()
            session.clear()
            val intent = Intent(requireContext(), LoginActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            }
            startActivity(intent)
        }
    }

    private fun stopLocationService() {
        val intent = Intent(requireContext(), LocationService::class.java).apply {
            action = LocationService.ACTION_STOP
        }
        requireContext().startService(intent)
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
