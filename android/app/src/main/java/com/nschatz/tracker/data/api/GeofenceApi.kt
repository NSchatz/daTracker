package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.CreateGeofenceRequest
import com.nschatz.tracker.data.model.Geofence
import com.nschatz.tracker.data.model.UpdateGeofenceRequest
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

interface GeofenceApi {
    @POST("geofences")
    suspend fun create(@Body request: CreateGeofenceRequest): Response<Geofence>

    @GET("geofences")
    suspend fun getAll(@Query("circle_id") circleId: String): Response<List<Geofence>>

    @PUT("geofences/{id}")
    suspend fun update(@Path("id") id: String, @Body request: UpdateGeofenceRequest): Response<Geofence>

    @DELETE("geofences/{id}")
    suspend fun delete(@Path("id") id: String): Response<Unit>
}
