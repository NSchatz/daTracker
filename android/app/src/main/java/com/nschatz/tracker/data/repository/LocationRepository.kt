package com.nschatz.tracker.data.repository

import com.nschatz.tracker.data.api.ApiClient
import com.nschatz.tracker.data.local.CachedLocationEntity
import com.nschatz.tracker.data.local.PendingLocationEntity
import com.nschatz.tracker.data.local.TrackerDatabase
import com.nschatz.tracker.data.model.LocationBatch
import com.nschatz.tracker.data.model.LocationPoint
import com.nschatz.tracker.data.model.MemberLocation
import com.nschatz.tracker.data.prefs.SessionManager

class LocationRepository(
    private val apiClient: ApiClient,
    private val sessionManager: SessionManager,
    private val database: TrackerDatabase
) {

    suspend fun submitLocations(circleId: String, points: List<LocationPoint>): Result<Unit> {
        return try {
            val response = apiClient.locations.postLocations(LocationBatch(circleId, points))
            if (response.isSuccessful) {
                Result.success(Unit)
            } else {
                Result.failure(Exception("Submit locations failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun queueLocation(entity: PendingLocationEntity) {
        database.pendingLocationDao().insert(entity)
    }

    suspend fun flushQueue(circleId: String): Result<Unit> {
        return try {
            val pending = database.pendingLocationDao().getOldest(100)
            if (pending.isEmpty()) return Result.success(Unit)

            val points = pending.map { entity ->
                LocationPoint(
                    lat = entity.lat,
                    lng = entity.lng,
                    speed = entity.speed,
                    batteryLevel = entity.batteryLevel,
                    accuracy = entity.accuracy,
                    recordedAt = entity.recordedAt
                )
            }
            val response = apiClient.locations.postLocations(LocationBatch(circleId, points))
            if (response.isSuccessful) {
                database.pendingLocationDao().deleteByIds(pending.map { it.id })
                Result.success(Unit)
            } else {
                Result.failure(Exception("Flush queue failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getLatest(circleId: String): Result<List<MemberLocation>> {
        return try {
            val response = apiClient.locations.getLatest(circleId)
            if (response.isSuccessful) {
                val body = response.body() ?: emptyList()
                val now = System.currentTimeMillis().toString()
                val cached = body.map { member ->
                    CachedLocationEntity(
                        orderId = "${member.userId}-latest",
                        userId = member.userId,
                        displayName = member.displayName,
                        lat = member.lat,
                        lng = member.lng,
                        batteryLevel = member.batteryLevel,
                        recordedAt = member.recordedAt,
                        updatedAt = now
                    )
                }
                database.cachedLocationDao().upsertAll(cached)
                Result.success(body)
            } else {
                Result.failure(Exception("Get latest failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getHistory(userId: String, from: String, to: String): Result<List<MemberLocation>> {
        return try {
            val response = apiClient.locations.getHistory(userId, from, to)
            if (response.isSuccessful) {
                Result.success(response.body() ?: emptyList())
            } else {
                Result.failure(Exception("Get history failed: ${response.code()} ${response.message()}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun pendingCount(): Int = database.pendingLocationDao().count()
}
