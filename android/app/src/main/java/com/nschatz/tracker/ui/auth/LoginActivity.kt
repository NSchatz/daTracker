package com.nschatz.tracker.ui.auth

import android.content.Intent
import android.os.Bundle
import android.view.View
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.prefs.SessionManager
import com.nschatz.tracker.data.repository.AuthRepository
import com.nschatz.tracker.databinding.ActivityLoginBinding
import com.nschatz.tracker.ui.main.MainActivity
import kotlinx.coroutines.launch

class LoginActivity : AppCompatActivity() {

    private lateinit var binding: ActivityLoginBinding
    private lateinit var sessionManager: SessionManager
    private lateinit var authRepo: AuthRepository

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        sessionManager = SessionManager(this)

        if (sessionManager.isLoggedIn) {
            goToMain()
            return
        }

        binding = ActivityLoginBinding.inflate(layoutInflater)
        setContentView(binding.root)

        authRepo = AuthRepository(ApiClient(sessionManager), sessionManager)

        binding.editServerUrl.setText(sessionManager.serverUrl)

        binding.btnLogin.setOnClickListener {
            val serverUrl = binding.editServerUrl.text?.toString()?.trim() ?: ""
            val email = binding.editEmail.text?.toString()?.trim() ?: ""
            val password = binding.editPassword.text?.toString() ?: ""

            if (serverUrl.isEmpty()) {
                showError("Server URL is required")
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

            sessionManager.serverUrl = serverUrl

            setLoading(true)
            lifecycleScope.launch {
                val result = AuthRepository(ApiClient(sessionManager), sessionManager).login(email, password)
                setLoading(false)
                result.fold(
                    onSuccess = { goToMain() },
                    onFailure = { showError(it.message ?: "Login failed") }
                )
            }
        }

        binding.btnRegister.setOnClickListener {
            startActivity(Intent(this, RegisterActivity::class.java))
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
        binding.btnLogin.isEnabled = !loading
        binding.btnRegister.isEnabled = !loading
        binding.txtError.visibility = View.GONE
    }

    private fun showError(message: String) {
        binding.txtError.text = message
        binding.txtError.visibility = View.VISIBLE
    }
}
