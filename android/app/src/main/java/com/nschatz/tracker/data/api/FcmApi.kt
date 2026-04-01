package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.FcmTokenRequest
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.POST

interface FcmApi {
    @POST("fcm-token")
    suspend fun registerToken(@Body request: FcmTokenRequest): Response<Unit>
}
