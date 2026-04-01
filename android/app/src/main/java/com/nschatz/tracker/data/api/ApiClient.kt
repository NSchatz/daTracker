package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.prefs.SessionManager
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

class ApiClient(private val sessionManager: SessionManager) {

    val rawOkHttpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    private val okHttpClient: OkHttpClient by lazy {
        val logging = HttpLoggingInterceptor().apply {
            level = HttpLoggingInterceptor.Level.BODY
        }
        OkHttpClient.Builder()
            .addInterceptor(TokenInterceptor(sessionManager))
            .addInterceptor(logging)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    private val retrofit: Retrofit by lazy {
        Retrofit.Builder()
            .baseUrl(sessionManager.serverUrl.trimEnd('/') + "/")
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    val auth: AuthApi by lazy { retrofit.create(AuthApi::class.java) }
    val locations: LocationApi by lazy { retrofit.create(LocationApi::class.java) }
    val circles: CircleApi by lazy { retrofit.create(CircleApi::class.java) }
    val geofences: GeofenceApi by lazy { retrofit.create(GeofenceApi::class.java) }
    val fcm: FcmApi by lazy { retrofit.create(FcmApi::class.java) }
}
