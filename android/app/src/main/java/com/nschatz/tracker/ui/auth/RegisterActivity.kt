package com.nschatz.tracker.ui.auth

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.AuthRepository
import com.nschatz.tracker.databinding.ActivityRegisterBinding
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.launch

class RegisterActivity : AppCompatActivity() {

    private lateinit var binding: ActivityRegisterBinding
    private lateinit var sessionManager: SessionManager
    private lateinit var authRepo: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        binding = ActivityRegisterBinding.inflate(layoutInflater)
        setContentView(binding.root)

        sessionManager = SessionManager(this)
        authRepo = AuthRepository(ApiClient(sessionManager), sessionManager)

        binding.btnRegister.setOnClickListener {
            val displayName = binding.editDisplayName.text?.toString()?.trim() ?: ""
            val email = binding.editEmail.text?.toString()?.trim() ?: ""
            val password = binding.editPassword.text?.toString() ?: ""
            val inviteCode = binding.editInviteCode.text?.toString()?.trim() ?: ""

            if (displayName.isEmpty()) {
                showError("Display name is required")
                return@setOnClickListener
            }
            if (email.isEmpty()) {
                showError("Email is required")
                return@setOnClickListener
            }
            if (password.isEmpty()) {
                showError("Password is required")
                return@setOnClickListener
            }
            if (inviteCode.isEmpty()) {
                showError("Invite code is required")
                return@setOnClickListener
            }

            setLoading(true)
            lifecycleScope.launch {
                val result = authRepo.register(email, password, displayName)
                setLoading(false)
                result.fold(
                    onSuccess = { goToMain() },
                    onFailure = { showError(it.message ?: "Registration failed") }
                )
            }
        }
    }

    private fun goToMain() {
        val intent = Intent(this, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        }
        startActivity(intent)
        finish()
    }

    private fun setLoading(loading: Boolean) {
        binding.progress.visibility = if (loading) View.VISIBLE else View.GONE
        binding.btnRegister.isEnabled = !loading
        binding.txtError.visibility = View.GONE
    }

    private fun showError(message: String) {
        binding.txtError.text = message
        binding.txtError.visibility = View.VISIBLE
    }
}
