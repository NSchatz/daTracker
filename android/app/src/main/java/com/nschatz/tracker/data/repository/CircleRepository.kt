package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.Circle
import com.nschatz.tracker.data.model.CircleMember
import com.nschatz.tracker.data.model.CreateCircleRequest
import com.nschatz.tracker.data.model.JoinCircleRequest

class CircleRepository(
    private val apiClient: ApiClient
) {

    suspend fun create(name: String): Result<Circle> {
        return try {
            val response = apiClient.circles.create(CreateCircleRequest(name))
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(Exception("Empty response body"))
                }
            } else {
                Result.failure(Exception("Create circle failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAll(): Result<List<Circle>> {
        return try {
            val response = apiClient.circles.getAll()
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get circles failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun join(circleId: String, inviteCode: String): Result<Unit> {
        return try {
            val response = apiClient.circles.join(circleId, JoinCircleRequest(inviteCode))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Join circle failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getMembers(circleId: String): Result<List<CircleMember>> {
        return try {
            val response = apiClient.circles.getMembers(circleId)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get members failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
