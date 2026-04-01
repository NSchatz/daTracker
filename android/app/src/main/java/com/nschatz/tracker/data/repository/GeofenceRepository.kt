package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.model.CreateGeofenceRequest
import com.nschatz.tracker.data.model.Geofence
import com.nschatz.tracker.data.model.UpdateGeofenceRequest

class GeofenceRepository(
    private val apiClient: ApiClient
) {

    suspend fun create(
        circleId: String,
        name: String,
        lat: Double,
        lng: Double,
        radiusMeters: Double
    ): Result<Geofence> {
        return try {
            val response = apiClient.geofences.create(
                CreateGeofenceRequest(circleId, name, lat, lng, radiusMeters)
            )
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(Exception("Empty response body"))
                }
            } else {
                Result.failure(Exception("Create geofence failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getAll(circleId: String): Result<List<Geofence>> {
        return try {
            val response = apiClient.geofences.getAll(circleId)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get geofences failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun update(id: String, name: String?, radiusMeters: Double?): Result<Geofence> {
        return try {
            val response = apiClient.geofences.update(id, UpdateGeofenceRequest(name, radiusMeters))
            if (response.isSuccessful) {
                val body = response.body()
                if (body != null) {
                    Result.success(body)
                } else {
                    Result.failure(Exception("Empty response body"))
                }
            } else {
                Result.failure(Exception("Update geofence failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun delete(id: String): Result<Unit> {
        return try {
            val response = apiClient.geofences.delete(id)
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Delete geofence failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
