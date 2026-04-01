package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.LocationBatch
import com.nschatz.tracker.data.model.MemberLocation
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Query

interface LocationApi {
    @POST("locations")
    suspend fun postLocations(@Body batch: LocationBatch): Response<Unit>

    @GET("locations/latest")
    suspend fun getLatest(@Query("circle_id") circleId: String): Response<List<MemberLocation>>

    @GET("locations/history")
    suspend fun getHistory(
        @Query("user_id") userId: String,
        @Query("from") from: String,
        @Query("to") to: String
    ): Response<List<MemberLocation>>
}
