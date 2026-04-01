package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.AuthResponse
import com.nschatz.tracker.data.model.LoginRequest
import com.nschatz.tracker.data.model.RegisterRequest
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface AuthApi {
    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthResponse>

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthResponse>
}
