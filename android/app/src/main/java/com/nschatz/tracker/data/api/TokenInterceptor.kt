package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.prefs.SessionManager
import okhttp3.Interceptor
import okhttp3.Response

class TokenInterceptor(private val sessionManager: SessionManager) : Interceptor {

    override fun intercept(chain: Interceptor.Chain): Response {
        val original = chain.request()
        val token = sessionManager.token
        return if (token != null) {
            val request = original.newBuilder()
                .header("Authorization", "Bearer $token")
                .build()
            chain.proceed(request)
        } else {
            chain.proceed(original)
        }
    }
}
