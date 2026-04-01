package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.LoginRequest
import com.nschatz.tracker.data.model.RegisterRequest
import com.nschatz.tracker.data.prefs.SessionManager

class AuthRepository(
    private val apiClient: ApiClient,
    private val sessionManager: SessionManager
) {

    suspend fun register(email: String, password: String, displayName: String): Result<Unit> {
        return try {
            val response = apiClient.auth.register(RegisterRequest(email, password, displayName))
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    sessionManager.saveAuth(body.token, body.user)
                    Result.success(Unit)
                } else {
                    Result.failure(Exception("Empty response body"))
                }
            } else {
                Result.failure(Exception("Registration failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun login(email: String, password: String): Result<Unit> {
        return try {
            val response = apiClient.auth.login(LoginRequest(email, password))
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    sessionManager.saveAuth(body.token, body.user)
                    Result.success(Unit)
                } else {
                    Result.failure(Exception("Empty response body"))
                }
            } else {
                Result.failure(Exception("Login failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    fun logout() {
        sessionManager.clear()
    }
}
