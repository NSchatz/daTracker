package com.nschatz.tracker.data.api

import com.nschatz.tracker.data.model.Circle
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.model.CreateCircleRequest
import com.nschatz.tracker.data.model.JoinCircleRequest
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path

interface CircleApi {
    @POST("circles")
    suspend fun create(@Body request: CreateCircleRequest): Response<Circle>

    @GET("circles")
    suspend fun getAll(): Response<List<Circle>>

    @POST("circles/{id}/join")
    suspend fun join(@Path("id") id: String, @Body request: JoinCircleRequest): Response<Unit>

    @GET("circles/{id}/members")
    suspend fun getMembers(@Path("id") id: String): Response<List<CircleMember>>
}
